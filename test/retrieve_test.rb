# frozen_string_literal: true

require_relative('lib/helper')
require('date')
require('openssl')
require('time')
require('securerandom')

class RetrieveTest < MiniTest::Test
  include(Helper::Include)

  def test_retrieve_day_failures_and_empty
    # invalid date
    resp = get_day('2020-30-01')
    assert_response(resp, 400, 'text/plain; charset=utf-8', body: "invalid date parameter\n")

    # Disallowed methods
    %w[post patch delete put].each do |meth|
      resp = get_day(yesterday_utc.iso8601, method: meth)
      assert_response(resp, 405, 'text/plain; charset=utf-8', body: "method not allowed\n")
    end

    # get empty data
    resp = get_day(yesterday_utc.iso8601)
    assert_response(resp, 200, 'application/x-protobuf; delimited=true')
    assert_equal(resp.headers['Cache-Control'], 'public, max-age=3600, max-stale=600')
    expect_no_keys(resp)
  end

  def test_retrieve_stuff
    active_at = time_in_date('10:00', today_utc.prev_day(8))
    add_key(active_at: active_at, submitted_at: time_in_date('03:00', yesterday_utc))
    rsn = rolling_start_number(active_at)

    start_time = yesterday_utc.to_datetime.to_time.to_i
    end_time = start_time + 86400

    resp = get_day(yesterday_utc.iso8601)
    assert_response(resp, 200, 'application/x-protobuf; delimited=true')
    expect_one_key(resp, rsn, 8, '1' * 16, start_time, end_time)
  end

  def test_24_hour_span
    a = time_in_date("10:00", today_utc.prev_day(13))
    two_days_ago = yesterday_utc.prev_day

    # Our retrieve-day endpoint returns keys SUBMITTED within the range of a day.
    # Here, we're making sure that that day is calculated in UTC, and that we
    # correctly manage the bounds of that date.
    add_key(active_at: a, submitted_at: time_in_date("23:59:59", two_days_ago), data: '1' * 16)
    add_key(active_at: a, submitted_at: time_in_date("00:00", yesterday_utc), data: '2' * 16)
    add_key(active_at: a, submitted_at: time_in_date("23:59:59", yesterday_utc), data: '3' * 16)
    add_key(active_at: a, submitted_at: time_in_date("00:00", today_utc), data: '4' * 16)

    rsn = rolling_start_number(a)

    start_time = yesterday_utc.to_datetime.to_time.to_i
    end_time = start_time + 86400

    resp = get_day(yesterday_utc.iso8601)
    assert_response(resp, 200, 'application/x-protobuf; delimited=true')
    expect_retrieve_data(
      resp,
      [
        Covidshield::File.new(
          header: Covidshield::Header.new(
            startTimestamp: start_time,
            endTimestamp: end_time,
            region: "ON",
            batchNum: 1,
            batchSize: 1
          ),
          key: [
            Covidshield::Key.new(
              keyData: "2222222222222222",
              rollingStartNumber: rsn,
              rollingPeriod: 144,
              transmissionRiskLevel: 8
            ),
            Covidshield::Key.new(
              keyData: "3333333333333333",
              rollingStartNumber: rsn,
              rollingPeriod: 144,
              transmissionRiskLevel: 8
            )
          ]
        )
      ]
    )
  end

  def test_14_day_window
    # Our 14ish-day window is calculated by taking the current timestamp, and
    # finding the current UTC date, then sutracting 14 days from that.]
    new = time_in_date("00:00", today_utc.prev_day(13))
    old = time_in_date("23:59", today_utc.prev_day(14))
    add_key(active_at: new, submitted_at: time_in_date("00:00", yesterday_utc))
    add_key(active_at: old, submitted_at: time_in_date("00:00", yesterday_utc), data: 'b' * 16)

    start_time = yesterday_utc.to_datetime.to_time.to_i
    end_time = start_time + 86400

    resp = get_day(yesterday_utc.iso8601)
    assert_response(resp, 200, 'application/x-protobuf; delimited=true')
    expect_one_key(resp, rolling_start_number(new), 8, '1' * 16, start_time, end_time)
  end

  def test_invalid_auth_for_day
    date = yesterday_utc
    hmac = OpenSSL::HMAC.hexdigest(
      "SHA256",
      [ENV.fetch("RETRIEVE_HMAC_KEY")].pack("H*"),
      "#{date.iso8601}:#{Time.now.to_i / 3600}"
    )

    # success
    resp = @ret_conn.get("/retrieve-day/#{date.iso8601}/#{hmac}")
    assert_response(resp, 200, 'application/x-protobuf; delimited=true')

    # hmac is keyed to date
    resp = @ret_conn.get("/retrieve-day/#{date.prev_day.iso8601}/#{hmac}")
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    # changing hmac breaks it
    resp = @ret_conn.get("/retrieve-day/#{date.iso8601}/11112222#{hmac[8..-1]}")
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    # hmac is required
    resp = @ret_conn.get("/retrieve-day/#{date.iso8601}")
    assert_response(resp, 404, 'text/plain; charset=utf-8', body: "404 page not found\n")
  end

  def test_invalid_auth_for_hour
    date = yesterday_utc
    hour = 5
    hmac = OpenSSL::HMAC.hexdigest(
      "SHA256",
      [ENV.fetch("RETRIEVE_HMAC_KEY")].pack("H*"),
      "#{date.iso8601}:#{format("%02d", hour)}:#{Time.now.to_i / 3600}"
    )

    # success
    resp = @ret_conn.get("/retrieve-hour/#{date.iso8601}/#{format("%02d", hour)}/#{hmac}")
    assert_response(resp, 200, 'application/x-protobuf; delimited=true')

    # hmac is keyed to date
    resp = @ret_conn.get("/retrieve-hour/#{date.prev_day.iso8601}/#{format("%02d", hour)}/#{hmac}")
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    # hmac is keyed to hour
    resp = @ret_conn.get("/retrieve-hour/#{date.prev_day.iso8601}/#{format("%02d", (hour+1)%24)}/#{hmac}")
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    # changing hmac breaks it
    resp = @ret_conn.get("/retrieve-hour/#{date.iso8601}/#{format("%02d", hour)}/11112222#{hmac[8..-1]}")
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    # hmac is required
    resp = @ret_conn.get("/retrieve-hour/#{date.iso8601}/#{format("%02d", hour)}")
    assert_response(resp, 404, 'text/plain; charset=utf-8', body: "404 page not found\n")
  end

  def test_reject_unacceptable_dates
    resp = get_day(today_utc.iso8601)
    assert_response(
      resp, 404, 'text/plain; charset=utf-8',
      body: "use /retrieve-hour for today's data\n"
    )

    resp = get_day(today_utc.next_day.iso8601)
    assert_response(
      resp, 404, 'text/plain; charset=utf-8',
      body: "cannot request future data\n"
    )

    resp = get_day(today_utc.prev_day(15).iso8601)
    assert_response(
      resp, 410, 'text/plain; charset=utf-8',
      body: "requested data no longer valid\n"
    )

    # TODO: Not 100% sure this is right. Should 14 or 13 be the oldest?
    resp = get_day(today_utc.prev_day(14).iso8601)
    assert_response(resp, 200, 'application/x-protobuf; delimited=true')
    expect_no_keys(resp)
  end

  def test_reject_unacceptable_hours
    resp = get_hour(today_utc.prev_day(2).iso8601, 23)
    assert_response(
      resp, 404, 'text/plain; charset=utf-8',
      body: "use /retrieve-day for data not from today or yesterday\n"
    )

    resp = get_hour(yesterday_utc.iso8601, 0)
    assert_response(resp, 200, 'application/x-protobuf; delimited=true')
    expect_no_keys(resp)

    resp = get_hour(today_utc.next_day.iso8601, 0)
    assert_response(
      resp, 404, 'text/plain; charset=utf-8',
      body: "use /retrieve-day for data not from today or yesterday\n"
    )

    resp = get_hour(today_utc.iso8601, 24)
    assert_response(
      resp, 400, 'text/plain; charset=utf-8',
      body: "invalid hour number\n"
    )

    now = Time.now.to_i
    current_hour = now / 3600 - 24 * (now / 86400)
    resp = get_hour(today_utc.iso8601, current_hour)
    assert_response(
      resp, 404, 'text/plain; charset=utf-8',
      body: "cannot serve data for current hour for privacy reasons\n"
    )
  end

  def test_500k
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

    assert_equal(%w(BC ON ON), files.map { |f| f.header.region }.sort)

    files.each.with_index do |file, index|
      assert_equal(start_time, file.header.startTimestamp)
      assert_equal(end_time, file.header.endTimestamp)
      assert_equal(index+1, file.header.batchNum)
      assert_equal(3, file.header.batchSize)

      key_data.concat(file.key.map(&:keyData))
    end

    assert_equal(18001, key_data.size)
    assert_equal(18001, key_data.uniq.size)
  end

  private

  def expect_one_key(resp, rsn, risk, data, start_time, end_time)
    expect_retrieve_data(
      resp,
      [
        Covidshield::File.new(
          header: Covidshield::Header.new(
            startTimestamp: start_time,
            endTimestamp: end_time,
            region: 'ON',
            batchNum: 1,
            batchSize: 1,
          ),
          key: [
            Covidshield::Key.new(
              keyData: data,
              rollingStartNumber: rsn,
              rollingPeriod: 144,
              transmissionRiskLevel: risk,
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

  def add_key(data: '1' * 16, active_at:, submitted_at:, risk_level: 8, region: 'ON')
    add_key_explicit(
      rsn: rolling_start_number(active_at),
      hour: hour_number(submitted_at),
      region: region,
      risk_level: risk_level,
      data: data
    )
  end

  def insert_key
    @insert_key ||= @dbconn.prepare(<<~SQL)
      INSERT INTO diagnosis_keys
      (key_data, rolling_start_number, rolling_period, risk_level, hour_of_submission, region)
      VALUES (?, ?, ?, ?, ?, ?)
    SQL
  end

  def add_key_explicit(data: '1' * 16, rsn:, risk_level: 8, hour:, region: 'ON', rolling_period: TEK_ROLLING_PERIOD)
    insert_key.execute(data, rsn, rolling_period, risk_level, hour, region)
  end

  def hour_number(timestamp)
    timestamp.to_i / 3600
  end

  TEK_ROLLING_PERIOD = 144

  def rolling_start_number(timestamp)
    en_interval_number = timestamp.to_i / 600
    (en_interval_number / TEK_ROLLING_PERIOD) * TEK_ROLLING_PERIOD
  end
end
