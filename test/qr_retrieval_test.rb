# frozen_string_literal: true

require_relative('lib/helper')
require('date')
require('openssl')
require('time')
require('securerandom')
require('stringio')
require('digest/sha2')
require('tempfile')



class QRRetrieveTest < MiniTest::Test
  include(Helper::Include)

  LOCATION_ID = 'ABCDEFGH'

  def assert_happy_zip_response(resp)
    assert_response(resp, 200, 'application/zip')
    assert_equal(resp.headers['Cache-Control'], 'public, max-age=3600, max-stale=600')
    export_proto, siglist_proto = extract_zip(resp.body)
    assert_valid_qr_signature_list(siglist_proto, export_proto)
    export = Covidshield::OutbreakEventExport.decode(export_proto)
    export
  end

  def test_retrieve_period_happy_path_no_keys
    dn = current_date_number - 3
    resp = get_qr_date(dn)
    export = assert_happy_zip_response(resp)
    assert_locations(export, [], date_number: dn)
  end

  def test_reject_unacceptable_periods
    
    # Test with feature flag
    config = get_app_config()
    resp = get_qr_date(current_date_number)

    if config["disableCurrentDateCheckFeatureFlag"]
      assert_response(resp, 200, 'application/zip')
    else
      assert_response(
        resp, 404, 'text/plain; charset=utf-8',
        body: "cannot serve data for current period for privacy reasons\n"
      )
    end

    resp = get_qr_date(current_date_number - 1)
    assert_response(resp, 200, 'application/zip')

    resp = get_qr_date(current_date_number + 1)
    assert_response(
      resp, 404, 'text/plain; charset=utf-8',
      body: "cannot request future data\n"
    )

    # almost too old
    resp = get_qr_date(current_date_number - 14)
    assert_response(resp, 200, 'application/zip')

    # too old
    resp = get_qr_date(current_date_number - 15)
    assert_response(resp, 410, 'text/plain; charset=utf-8', body: "requested data no longer valid\n")
  end

  def test_disallowed_methods
    # Disallowed methods
    %w[post patch delete put].each do |meth|
      resp = get_qr_date(current_date_number - 1, method: meth)
      assert_response(resp, 405, 'text/plain; charset=utf-8', body: "method not allowed\n")
    end
  end

  def test_retrieve_stuff
    start_time = time_in_date('10:00', today_utc.prev_day(8))
    end_time = time_in_date('12:00', today_utc.prev_day(8))
    add_location(start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date('07:00', yesterday_utc))

    dn = current_date_number - 1

    resp = get_qr_date(dn)
    export = assert_happy_zip_response(resp)
    locations = [location(
        start_time: start_time,
        end_time: end_time,
        severity: 1
    )]
    assert_locations(export, locations, date_number: dn)
  end

  def test_all_keys
    start_time = time_in_date('10:00', today_utc.prev_day(8))
    end_time = time_in_date('12:00', today_utc.prev_day(8))
    two_days_ago = yesterday_utc.prev_day(1)
    fourteen_days_ago = yesterday_utc.prev_day(14)
    fiveteen_days_ago = yesterday_utc.prev_day(15)

    # Our retrieve endpoint returns keys CREATED within the given period.
    add_location(location_id: '1' * 8, start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date("23:59:59", fiveteen_days_ago))
    add_location(location_id: '2' * 8, start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date("00:00", fourteen_days_ago))
    add_location(location_id: '3' * 8, start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date("01:59:59", yesterday_utc))
    add_location(location_id: '4' * 8, start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date("02:00", yesterday_utc))
    add_location(location_id: '5' * 8, start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date("02:00", yesterday_utc))
    add_location(location_id: '6' * 8, start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date("02:00", today_utc))

    resp = get_qr_date("00000")

    config = get_app_config()

    if config["enableEntirePeriodBundle"]
      export = assert_happy_zip_response(resp)
      locations = [location(
        location_id: '2' * 8,
        start_time: start_time,
        end_time: end_time,
        severity: 1,
    ), location(
        location_id: '3' * 8,
        start_time: start_time,
        end_time: end_time,
        severity: 1,
    ), location(
        location_id: '4' * 8,
        start_time: start_time,
        end_time: end_time,
        severity: 1,
    ), location(
        location_id: '5' * 8,
        start_time: start_time,
        end_time: end_time,
        severity: 1,
    )]
      assert_equal(locations, export.locations)
    else
      assert_response(resp, 410, 'text/plain; charset=utf-8', body: "requested data no longer valid\n")
    end
  end

  def test_period_bounds
    start_time = time_in_date('10:00', today_utc.prev_day(8))
    end_time = time_in_date('12:00', today_utc.prev_day(8))
    two_days_ago = yesterday_utc.prev_day(1)

    # Our retrieve endpoint returns keys CREATED within the given period.
    add_location(location_id: '1' * 8, start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date("23:59:59", two_days_ago))
    add_location(location_id: '2' * 8, start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date("00:00", yesterday_utc))
    add_location(location_id: '3' * 8, start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date("01:59:59", yesterday_utc))
    add_location(location_id: '4' * 8, start_time: start_time.to_i, end_time: end_time.to_i, severity: 1, created: time_in_date("02:00", yesterday_utc))

    dn = current_date_number - 1

    resp = get_qr_date(dn)
    export = assert_happy_zip_response(resp)
    locations = [location(
        location_id: '2' * 8,
        start_time: start_time,
        end_time: end_time,
        severity: 1,
    ), location(
        location_id: '3' * 8,
        start_time: start_time,
        end_time: end_time,
        severity: 1,
    ), location(
        location_id: '4' * 8,
        start_time: start_time,
        end_time: end_time,
        severity: 1,
    )]
    assert_equal(locations, export.locations)
  end

  def test_invalid_auth
    dn = current_date_number - 2
    hmac = OpenSSL::HMAC.hexdigest(
      "SHA256",
      [ENV.fetch("RETRIEVE_HMAC_KEY")].pack("H*"),
      "302:#{dn}:#{Time.now.to_i / 3600}"
    )

    # success
    resp = @ret_conn.get("/qr/302/#{dn}/#{hmac}")
    assert_response(resp, 200, 'application/zip')

    # hmac is keyed to date
    resp = @ret_conn.get("/qr/302/#{dn - 1}/#{hmac}")
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    # changing hmac breaks it
    resp = @ret_conn.get("/qr/302/#{dn}/11112222#{hmac[8..-1]}")
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    # hmac is required
    resp = @ret_conn.get("/qr/302/#{dn}")
    assert_response(resp, 404, 'text/plain; charset=utf-8', body: "404 page not found\n")
  end

  private

  def expect_no_keys(resp)
    @buf = resp.body.each_byte.to_a
    files = load_retrieve_stream
    assert_equal([], files, "  (from #{caller[0]})")
  end

  def add_location(location_id: LOCATION_ID, originator: "ON", start_time: Time.now.to_i, end_time: Time.now.to_i, created: Time.now, severity: 1)
    insert_location.execute(location_id, originator, start_time, end_time, created, severity)
  end

  def insert_location
    @insert_location ||= @dbconn.prepare(<<~SQL)
      INSERT INTO qr_outbreak_events
      (location_id, originator, start_time, end_time, created, severity)
      VALUES (?, ?, ?, ?, ?, ?)
    SQL
  end

  def assert_valid_signature(signature, data)
    key_hex = ENV.fetch('ECDSA_KEY')
    key_der = [key_hex].pack('H*')
    key = OpenSSL::PKey::EC.new(key_der)
    key.check_key
    digest = Digest::SHA256.digest(data)
    key.dsa_verify_asn1(digest, signature)
  end

  def assert_locations(export, locations, date_number:)
    start_time = date_number * 86400
    end_time = (date_number + 1) * 86400

    assert_equal(
      Covidshield::OutbreakEventExport.new(
        start_timestamp: start_time,
        end_timestamp: end_time,
        locations: locations
      ).to_json, export.to_json
    )
  end

  def assert_valid_qr_signature_list(siglist_proto, export_proto)
    siglist = Covidshield::OutbreakEventExportSignature.decode(siglist_proto)
    assert_valid_signature(siglist.signature, export_proto)

    assert_equal(
      Covidshield::OutbreakEventExportSignature.new(
        signature: siglist.signature
      ).to_json, siglist.to_json
    )
  end

  def location(location_id: LOCATION_ID, start_time: Time.now, end_time: Time.now, severity: 1)
    Covidshield::OutbreakEvent.new(
      location_id: location_id,
      start_time: start_time,
      end_time: end_time,
      severity: severity,
    )
  end
end
