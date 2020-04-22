# frozen_string_literal: true

require_relative('lib/helper')
require('rbnacl')
require('set')

class RoundtripTest < MiniTest::Test
  include(Helper::Include)

  def test_key_roundtrip
    keys = (0...14).map do |i|
      tek(data: '1' * 15 + (i+97).chr, rolling_period: 144, rolling_start_number: current_rsn - 144 * i)
    end
    first_keys = keys.dup

    payload = Covidshield::Upload.new(timestamp: Time.now, keys: keys).to_proto

    credentials = new_valid_keyset

    resp = @sub_conn.post('/upload', encrypted_request(payload, credentials).to_proto)
    assert_result(resp, 200, :NONE, 14)
    expect_keys([]) # no visible keys because these are still in the current hour

    move_forward_days(1) # total: +1 days

    # Replace one of the 14 keys with a "new" one
    keys.pop
    keys.unshift(tek(data: '1' * 15 + 'z', rolling_start_number: current_rsn))
    payload = Covidshield::Upload.new(timestamp: Time.now, keys: keys).to_proto

    resp = @sub_conn.post('/upload', encrypted_request(payload, credentials).to_proto)
    assert_result(resp, 200, :NONE, 15)
    expect_keys(first_keys[0..-2])

    move_forward_hours(1) # total: +1 day & 1 hour
    expect_keys(first_keys[0..-2] + [keys.first])

    move_forward_hours(12 * 24 + 22) # total: +13 days & 23 hours
    expect_keys(first_keys[0..0] + [keys.first])

    resp = @sub_conn.post('/upload', encrypted_request(payload, credentials).to_proto)
    assert_result(resp, 200, :NONE, 15)
    expect_keys(first_keys[0..0] + [keys.first])

    move_forward_hours(1) # total: +14 days

    # In this range, the credentials could be valid or invalid, depending on
    # how far we were into the UTC date when we created the keypair.

    # We don't try hard to decide whether or not this is valid, but the
    # application should only be uploading credentials on days T+[0,13] after
    # diagnosis. (i.e. 14 total days, starting on diagnosis day)

    move_forward_days(1) # total: +15 days

    resp = @sub_conn.post('/upload', encrypted_request(payload, credentials).to_proto)
    assert_result(resp, 401, :INVALID_KEYPAIR, 15)
    expect_keys([])
  end

  private

  def expect_keys(want_keys)
    keys = []

    # inclusive: ask for the 14th day prior
    (1..14).each do |days_ago|
      resp = get_day(today_utc.prev_day(days_ago))
      assert_response(resp, 200, 'application/x-protobuf; delimited=true')
      keys.concat(parse_keys(resp))
    end

    # exclusive: don't ask for the current hour
    (0...(Time.now.utc.hour)).each do |hour|
      resp = get_hour(today_utc, hour)
      assert_response(resp, 200, 'application/x-protobuf; delimited=true')
      keys.concat(parse_keys(resp))
    end

    have_key_ids =      keys.map { |k| k.keyData[-1] }.sort
    want_key_ids = want_keys.map { |k| k.keyData[-1] }.sort
    assert_equal(want_key_ids, have_key_ids, "  (from #{caller[0]})")
  end

  def parse_keys(resp)
    keys = []
    @buf = resp.body.each_byte.to_a
    load_retrieve_stream.flat_map(&:key)
  end

  def count_diagnosis_keys
    @dbconn.query("SELECT COUNT(*) FROM diagnosis_keys").first.values.first
  end

  def current_rsn(ts: Time.now)
    (ts.to_i / 86400) * 144
  end

  def dummy_payload(nkeys=1)
    Covidshield::Upload.new(timestamp: Time.now, keys: [tek]*nkeys).to_proto
  end

  def encrypted_request(
    payload, keyset, server_public: keyset[:server_public], app_private: keyset[:app_private],
    app_public: keyset[:app_public], app_public_to_send: app_public,
    server_public_to_send: server_public,
    box: RbNaCl::Box.new(server_public, app_private),
    nonce: RbNaCl::Random.random_bytes(box.nonce_bytes),
    nonce_to_send: nonce,
    encrypted_payload: box.encrypt(nonce, payload)
  )
    Covidshield::EncryptedUploadRequest.new(
      serverPublicKey: server_public_to_send.to_s,
      appPublicKey: app_public_to_send.to_s,
      nonce: nonce_to_send,
      payload: encrypted_payload,
    )
  end

  def tek(data: '1' * 16, risk_level: 3, rolling_period: 144, rolling_start_number: 1234)
    Covidshield::Key.new(
      keyData: data,
      transmissionRiskLevel: risk_level,
      rollingPeriod: rolling_period,
      rollingStartNumber: rolling_start_number
    )
  end

  def assert_result(resp, code, error, keys)
    assert_response(resp, code, 'application/x-protobuf')
    assert_equal(
      Covidshield::EncryptedUploadResponse.new(error: error),
      Covidshield::EncryptedUploadResponse.decode(resp.body)
    )
    assert_equal(keys, count_diagnosis_keys)
  end
end
