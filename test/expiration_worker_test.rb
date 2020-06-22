# frozen_string_literal: true

require_relative('lib/helper')
require('rbnacl')
require('set')

class ExpirationWorkerTest < MiniTest::Test
  include(Helper::Include)

  def test_one_time_code_expiration
    req = encrypted_request(dummy_payload, new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 200, :NONE)

    expire_and_assert(encryption: 1, diagnosis: 1)

    new_valid_one_time_code
    expire_and_assert(encryption: 2, diagnosis: 1)

    move_forward_seconds(1440 * 60 - 2) # T+23:59:58
    expire_and_assert(encryption: 2, diagnosis: 1)

    move_forward_seconds(4) # T+24:00:02
    expire_and_assert(encryption: 1, diagnosis: 1)
  end

  def test_diagnosis_key_expiration
    req = encrypted_request(dummy_payload, new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 200, :NONE)

    expire_and_assert(encryption: 1, diagnosis: 1)

    would_expire_now = today_utc.prev_day(15).to_datetime.to_time.to_i
    expiry_hour = would_expire_now / 3600

    # Doesn't expire if it's more recent
    @dbconn.query("UPDATE diagnosis_keys SET hour_of_submission = #{expiry_hour + 1}")
    expire_and_assert(diagnosis: 1)

    # Doesn't expire if it's the last one allowed
    @dbconn.query("UPDATE diagnosis_keys SET hour_of_submission = #{expiry_hour}")
    expire_and_assert(diagnosis: 1)

    # Expires
    @dbconn.query("UPDATE diagnosis_keys SET hour_of_submission = #{expiry_hour - 1}")
    expire_and_assert(diagnosis: 0)
  end

  def test_claimed_encryption_key_expiration
    req = encrypted_request(dummy_payload, new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 200, :NONE)

    seconds_since_utc_midnight = Time.now.to_i % 86400
    hours_until_utc_midnight = 24 - (seconds_since_utc_midnight / 3600)

    move_forward_hours(14 * 24)
    move_forward_hours(hours_until_utc_midnight - 1)
    expire_and_assert(encryption: 1)

    move_forward_hours(1)
    expire_and_assert(encryption: 0)
  end

  private

  def expire
    # this doesn't yield until the server returns ok, which doesn't happen
    # until after the first run of the expiration worker.
    Helper.with_server(KEY_RETRIEVAL_SERVER, RETRIEVAL_SERVER_ADDR) { }
  end

  def expire_and_assert(encryption: nil, diagnosis: nil)
    expire
    assert_equal(diagnosis, count_diagnosis_keys, "  (from #{caller[0]})") if diagnosis
    assert_equal(encryption, count_encryption_keys, "  (from #{caller[0]})") if encryption
  end

  def count_diagnosis_keys
    @dbconn.query("SELECT COUNT(*) FROM diagnosis_keys").first.values.first
  end

  def count_encryption_keys
    @dbconn.query("SELECT COUNT(*) FROM encryption_keys").first.values.first
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

  def assert_result(resp, code, error)
    assert_response(resp, code, 'application/x-protobuf')
    assert_equal(
      Covidshield::EncryptedUploadResponse.new(error: error),
      Covidshield::EncryptedUploadResponse.decode(resp.body)
    )
  end
end
