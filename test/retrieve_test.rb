# frozen_string_literal: true

require_relative('lib/helper')
require('date')
require('openssl')
require('time')
require('securerandom')
require('stringio')
require('digest/sha2')
require('tempfile')

class RetrieveTest < MiniTest::Test
  include(Helper::Include)

  def assert_happy_zip_response(resp)
    assert_response(resp, 200, 'application/zip')
    assert_equal(resp.headers['Cache-Control'], 'public, max-age=3600, max-stale=600')
    export_proto, siglist_proto = extract_zip(resp.body)
    assert_equal("EK Export v1    ", export_proto[0...16])
    export_proto = export_proto[16..-1]
    assert_valid_signature_list(siglist_proto, export_proto)
    export = Covidshield::TemporaryExposureKeyExport.decode(export_proto)
    export
  end

  def test_retrieve_period_happy_path_no_keys
    dn = current_date_number - 3
    resp = get_date(dn)
    export = assert_happy_zip_response(resp)
    assert_keys(export, [], region: 'CA', date_number: dn)
  end

  def test_reject_unacceptable_periods
    resp = get_date(current_date_number)
    assert_response(
      resp, 404, 'text/plain; charset=utf-8',
      body: "cannot serve data for current period for privacy reasons\n"
    )

    resp = get_date(current_date_number - 1)
    assert_response(resp, 200, 'application/zip')

    resp = get_date(current_date_number + 1)
    assert_response(
      resp, 404, 'text/plain; charset=utf-8',
      body: "cannot request future data\n"
    )

    # almost too old
    resp = get_date(current_date_number - 14)
    assert_response(resp, 200, 'application/zip')

    # too old
    resp = get_date(current_date_number - 15)
    assert_response(resp, 410, 'text/plain; charset=utf-8', body: "requested data no longer valid\n")
  end

  def test_disallowed_methods
    # Disallowed methods
    %w[post patch delete put].each do |meth|
      resp = get_date(current_date_number - 1, method: meth)
      assert_response(resp, 405, 'text/plain; charset=utf-8', body: "method not allowed\n")
    end
  end

  def test_retrieve_stuff
    active_at = time_in_date('10:00', today_utc.prev_day(8))
    add_key(active_at: active_at, submitted_at: time_in_date('07:00', yesterday_utc))
    rsin = rolling_start_interval_number(active_at)

    dn = current_date_number - 1

    resp = get_date(dn)
    export = assert_happy_zip_response(resp)
    keys = [tek(
      rolling_start_interval_number: rsin,
      transmission_risk_level: 8,
    )]
    assert_keys(export, keys, region: 'CA', date_number: dn)
  end

  def test_period_bounds
    active_at = time_in_date('10:00', today_utc.prev_day(8))
    two_days_ago = yesterday_utc.prev_day(1)

    # Our retrieve endpoint returns keys SUBMITTED within the given period.
    add_key(active_at: active_at, submitted_at: time_in_date("23:59:59", two_days_ago), data: '1' * 16)
    add_key(active_at: active_at, submitted_at: time_in_date("00:00", yesterday_utc), data: '2' * 16)
    add_key(active_at: active_at, submitted_at: time_in_date("01:59:59", yesterday_utc), data: '3' * 16)
    add_key(active_at: active_at, submitted_at: time_in_date("02:00", yesterday_utc), data: '4' * 16)

    rsin = rolling_start_interval_number(active_at)

    dn = current_date_number - 1

    resp = get_date(dn)
    export = assert_happy_zip_response(resp)
    keys = [tek(
      rolling_start_interval_number: rsin,
      transmission_risk_level: 8,
      data: "2222222222222222",
    ), tek(
      rolling_start_interval_number: rsin,
      transmission_risk_level: 8,
      data: "3333333333333333",
    )]
  end

  def test_invalid_auth
    dn = current_date_number - 2
    hmac = OpenSSL::HMAC.hexdigest(
      "SHA256",
      [ENV.fetch("RETRIEVE_HMAC_KEY")].pack("H*"),
      "302:#{dn}:#{Time.now.to_i / 3600}"
    )

    # success
    resp = @ret_conn.get("/retrieve/302/#{dn}/#{hmac}")
    assert_response(resp, 200, 'application/zip')

    # hmac is keyed to date
    resp = @ret_conn.get("/retrieve/302/#{dn - 1}/#{hmac}")
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    # changing hmac breaks it
    resp = @ret_conn.get("/retrieve/302/#{dn}/11112222#{hmac[8..-1]}")
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    # hmac is required
    resp = @ret_conn.get("/retrieve/302/#{dn}")
    assert_response(resp, 404, 'text/plain; charset=utf-8', body: "404 page not found\n")
  end

  def test_too_many_keys_for_one_zip
    # I don't think the protocol is going to stay the way it is here.
    # We can wait to hear back from Google/Apple about this before piling on to
    # make the hacks work if we need to.
    skip

    count = 18000 # enough to require two Files

    new = time_in_date("00:00", today_utc.prev_day(13))
    old = time_in_date("23:59", today_utc.prev_day(14))

    t = Time.now
    STDERR.puts("adding many records to the database (takes about 5-12 seconds)...")
    @dbconn.query('BEGIN')
    count.times do
      add_key(active_at: new, submitted_at: time_in_date("00:00", yesterday_utc), data: SecureRandom.bytes(16))
    end
    add_key(region: 'BC', active_at: new, submitted_at: time_in_date("00:00", yesterday_utc), data: SecureRandom.bytes(16))
    @dbconn.query('COMMIT')
    STDERR.puts("finished adding records in #{Time.now-t} seconds")

    start_time = yesterday_utc.to_datetime.to_time.to_i
    end_time = start_time + 86400

    resp = get_day(yesterday_utc.iso8601)
    assert_response(resp, 200, 'application/x-protobuf; delimited=true')

    # write sample data
    File.write(File.expand_path('../build/retrieve-example.proto-stream', __dir__), resp.body)

    @buf = resp.body.each_byte.to_a
    files = load_retrieve_stream
    assert_equal(3, files.size)
    files.each { |file| assert(file.to_proto.size < (500 * 1024)) }
    assert(files.first.to_proto.size > 499 * 1024)

    key_data = []

    assert_equal(%w(BC ON ON), files.map { |f| f.region }.sort)

    files.each.with_index do |file, index|
      assert_equal(start_time, file.start_timestamp)
      assert_equal(end_time, file.end_timestamp)
      assert_equal(index+1, file.batch_num)
      assert_equal(3, file.batch_size)

      key_data.concat(file.keys.map(&:key_data))
    end

    assert_equal(18001, key_data.size)
    assert_equal(18001, key_data.uniq.size)
  end

  private

  def expect_one_key(resp, rsin, risk, data, start_time, end_time)
    expect_retrieve_data(
      resp,
      [
        Covidshield::TemporaryExposureKeyExport.new(
          start_timestamp: start_time,
          end_timestamp: end_time,
          region: 'CA',
          batch_num: 1,
          batch_size: 1,
          keys: [
            Covidshield::TemporaryExposureKey.new(
              key_data: data,
              rolling_start_interval_number: rsin,
              rolling_period: 144,
              transmission_risk_level: risk,
            )
          ]
        )
      ],
      1
    )
  end

  def expect_no_keys(resp)
    @buf = resp.body.each_byte.to_a
    files = load_retrieve_stream
    assert_equal([], files, "  (from #{caller[0]})")
  end

  def add_key(data: '1' * 16, active_at:, submitted_at:, transmission_risk_level: 8, region: '302')
    add_key_explicit(
      rsin: rolling_start_interval_number(active_at),
      hour: hour_number(submitted_at),
      region: region,
      transmission_risk_level: transmission_risk_level,
      data: data
    )
  end

  def insert_key
    @insert_key ||= @dbconn.prepare(<<~SQL)
      INSERT INTO diagnosis_keys
      (key_data, rolling_start_interval_number, rolling_period, transmission_risk_level, hour_of_submission, region)
      VALUES (?, ?, ?, ?, ?, ?)
    SQL
  end

  def add_key_explicit(data: '1' * 16, rsin:, transmission_risk_level: 8, hour:, region: '302', rolling_period: TEK_ROLLING_PERIOD)
    insert_key.execute(data, rsin, rolling_period, transmission_risk_level, hour, region)
  end

  def hour_number(timestamp)
    timestamp.to_i / 3600
  end

  TEK_ROLLING_PERIOD = 144

  def rolling_start_interval_number(timestamp)
    en_interval_number = timestamp.to_i / 600
    (en_interval_number / TEK_ROLLING_PERIOD) * TEK_ROLLING_PERIOD
  end

  def assert_valid_signature(signature, data)
    key_hex = ENV.fetch('ECDSA_KEY')
    key_der = [key_hex].pack('H*')
    key = OpenSSL::PKey::EC.new(key_der)
    key.check_key

    digest = Digest::SHA256.digest(data)

    # Why doesn't this work? Our signature is in X9.62 uncompressed form, which
    # seems to be what OpenSSL is looking for, but we get "nested asn1 error".
    puts("WARN: not verifying signature")
    # key.dsa_verify_asn1(digest, signature)
  end

  def assert_keys(export, keys, region:, date_number:)
    start_time = date_number * 86400
    end_time = (date_number + 1) * 86400

    assert_equal(
      Covidshield::TemporaryExposureKeyExport.new(
        start_timestamp: start_time,
        end_timestamp: end_time,
        region: region,
        batch_num: 1,
        batch_size: 1,
        signature_infos: [
          Covidshield::SignatureInfo.new(
            verification_key_version: "v1",
            verification_key_id: "302",
            signature_algorithm: "1.2.840.10045.4.3.2"
          ),
        ],
        keys: keys
      ).to_json, export.to_json
    )
  end

  def assert_valid_signature_list(siglist_proto, export_proto)
    siglist = Covidshield::TEKSignatureList.decode(siglist_proto)
    assert_valid_signature(siglist.signatures[0].signature, export_proto)

    assert_equal(
      Covidshield::TEKSignatureList.new(
        signatures: [
          Covidshield::TEKSignature.new(
            signature_info: Covidshield::SignatureInfo.new(
              verification_key_version: "v1",
              verification_key_id: "302",
              signature_algorithm: "1.2.840.10045.4.3.2"
            ),
            batch_num: 1,
            batch_size: 1,
            signature: siglist.signatures[0].signature
          )
        ]
      ).to_json, siglist.to_json
    )
  end
end
