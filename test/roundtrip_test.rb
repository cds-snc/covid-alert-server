# frozen_string_literal: true

require_relative('lib/helper')
require('rbnacl')
require('set')

class RoundtripTest < MiniTest::Test
  include(Helper::Include)

  def days_ago(n)
    current_rsin - 144 * n
  end

  def days_ago_at_noon(n)
    ((Date.today.to_time.to_i + (12 * 60 * 60)) - (n * 86400)) / (60 * 10)
  end

  def test_key_roundtrip
    puts days_ago_at_noon(1)
    puts days_ago(1)
    keys = [
      tek(data: '111111111111111a', rolling_start_interval_number: days_ago_at_noon(1)),
      tek(data: '111111111111111b', rolling_start_interval_number: days_ago_at_noon(2)),
      tek(data: '111111111111111c', rolling_start_interval_number: days_ago_at_noon(3)),
      tek(data: '111111111111111d', rolling_start_interval_number: days_ago_at_noon(4)),
      tek(data: '111111111111111e', rolling_start_interval_number: days_ago_at_noon(5)),
      tek(data: '111111111111111f', rolling_start_interval_number: days_ago_at_noon(6)),
      tek(data: '111111111111111g', rolling_start_interval_number: days_ago_at_noon(7)),
      tek(data: '111111111111111h', rolling_start_interval_number: days_ago_at_noon(8)),
      tek(data: '111111111111111i', rolling_start_interval_number: days_ago_at_noon(9)),
      tek(data: '111111111111111j', rolling_start_interval_number: days_ago_at_noon(10)),
      tek(data: '111111111111111k', rolling_start_interval_number: days_ago_at_noon(11)),
      tek(data: '111111111111111l', rolling_start_interval_number: days_ago_at_noon(12)),
      tek(data: '111111111111111m', rolling_start_interval_number: days_ago_at_noon(13)),
    ]
    first_keys = keys.dup

    payload = Covidshield::Upload.new(timestamp: Time.now, keys: keys).to_proto

    credentials = new_valid_keyset

    resp = @sub_conn.post('/upload', encrypted_request(payload, credentials).to_proto)
    assert_result(resp, 200, :NONE, 13)
    expect_keys([]) # no visible keys because these are still in the current day

    move_forward_days(1) # total: +1 days

    # Replace one of the 14 keys with a "new" one
    keys.pop
    keys.each { |k| k.rolling_start_interval_number -= 144 }
    keys.unshift(tek(data: '111111111111111z', rolling_start_interval_number: days_ago_at_noon(1)))
    payload = Covidshield::Upload.new(timestamp: Time.now, keys: keys).to_proto

    resp = @sub_conn.post('/upload', encrypted_request(payload, credentials).to_proto)
    assert_result(resp, 200, :NONE, 14)
    # expected ["a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m"]
    # actual   [     "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m"]
    expect_keys(first_keys[0..-1]) ###############################################################################################################################

    move_forward_hours(3) # total: +1 day & 1 hour
    expect_keys(first_keys[0..-1])

    move_forward_hours(1) # total: +1 day & 2 hours
    expect_keys(first_keys[0..-1])

    move_forward_hours(11 * 24 + 21) # total: +12 days & 23 hours
    expect_keys(first_keys[0..0] + [keys.first])

    resp = @sub_conn.post('/upload', encrypted_request(payload, credentials).to_proto)
    assert_result(resp, 200, :NONE, 14)
    expect_keys(first_keys[0..0] + [keys.first])

    move_forward_hours(1) # total: +13 days
    expect_keys(first_keys[0..0] + [keys.first])

    # In this range, the credentials could be valid or invalid, depending on
    # how far we were into the UTC date when we created the keypair.

    # We don't try hard to decide whether or not this is valid, but the
    # application should only be uploading credentials on days T+[0,13] after
    # diagnosis. (i.e. 14 total days, starting on diagnosis day)

    move_forward_days(1) # total: +14 days
    assert_result(resp, 200, :NONE, 14)
    expect_keys([keys.first])

    move_forward_days(1) # total: +15 days

    resp = @sub_conn.post('/upload', encrypted_request(payload, credentials).to_proto)
    assert_result(resp, 401, :INVALID_KEYPAIR, 14)
    expect_keys([])
  end

  private

  def expect_keys(want_keys)
    keys = []

    number_of_periods = 14
    number_of_periods.times do |n|
      dn = current_date_number - (1 + n)
      resp = get_date(dn)
      assert_response(resp, 200, 'application/zip')
      keys.concat(parse_keys(resp))
    end

    have_key_ids =      keys.map { |k| k.key_data[-1] }.sort
    want_key_ids = want_keys.map { |k| k.key_data[-1] }.sort
    assert_equal(want_key_ids, have_key_ids, "  (from #{caller[0]})")
  end

  def parse_keys(resp)
    export_proto, siglist_proto = extract_zip(resp.body)
    export = Covidshield::TemporaryExposureKeyExport.decode(export_proto[16..-1])
    export.keys
  end

  def count_diagnosis_keys
    @dbconn.query("SELECT COUNT(*) FROM diagnosis_keys").first.values.first
  end

  def current_rsin(ts: Time.now)
    (ts.to_i / 86400) / (60 * 10)
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
      server_public_key: server_public_to_send.to_s,
      app_public_key: app_public_to_send.to_s,
      nonce: nonce_to_send,
      payload: encrypted_payload,
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