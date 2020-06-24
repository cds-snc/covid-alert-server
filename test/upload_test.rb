# frozen_string_literal: true

require_relative('lib/helper')
require('rbnacl')

class UploadTest < MiniTest::Test
  include(Helper::Include)

  def test_upload_test
    # too much post data
    resp = @sub_conn.post('/upload', 'a'*2000)
    assert_result(resp, 400, :UNKNOWN)

    # incomprehensible post data
    resp = @sub_conn.post('/upload', 'a'*100)
    assert_result(resp, 400, :UNKNOWN)

    # happy path
    req = encrypted_request(dummy_payload, new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 200, :NONE)
    assert_equal(['first-token'], diagnosis_originators)

    # timestamp almost too old
    req = encrypted_request(dummy_payload(timestamp: Time.at(Time.now.to_i - 59*60)), new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 200, :NONE)

    # timestamp too old
    req = encrypted_request(dummy_payload(timestamp: Time.at(Time.now.to_i - 3600)), new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :INVALID_TIMESTAMP)

    # app_public too long
    req = encrypted_request(dummy_payload, new_valid_keyset, app_public_to_send:("a"*50))
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :INVALID_CRYPTO_PARAMETERS)

    # app_public too short
    req = encrypted_request(dummy_payload, new_valid_keyset, app_public_to_send:("a"*4))
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :INVALID_CRYPTO_PARAMETERS)

    # server_public too short
    req = encrypted_request(dummy_payload, new_valid_keyset, server_public_to_send:("a"*4))
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :INVALID_CRYPTO_PARAMETERS)

    # server_public too long
    req = encrypted_request(dummy_payload, new_valid_keyset, server_public_to_send:("a"*50))
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :INVALID_CRYPTO_PARAMETERS)

    # server_public doesn't resolve to a server_private
    req = encrypted_request(dummy_payload, new_valid_keyset, server_public_to_send:("a"*32))
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 401, :INVALID_KEYPAIR)

    # server_public too short
    req = encrypted_request(dummy_payload, new_valid_keyset, server_public_to_send:("a"*4))
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :INVALID_CRYPTO_PARAMETERS)

    # server_public too long
    req = encrypted_request(dummy_payload, new_valid_keyset, server_public_to_send:("a"*50))
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :INVALID_CRYPTO_PARAMETERS)

    # nonce too long
    req = encrypted_request(dummy_payload, new_valid_keyset, nonce_to_send:("a"*23))
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :INVALID_CRYPTO_PARAMETERS)

    # nonce too long
    req = encrypted_request(dummy_payload, new_valid_keyset, nonce_to_send:("a"*32))
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :INVALID_CRYPTO_PARAMETERS)

    # invalid encrypted payload
    req = encrypted_request(dummy_payload, new_valid_keyset, encrypted_payload:("a"*64))
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :DECRYPTION_FAILED)

    # invalid contents of valid encrypted payload
    req = encrypted_request("aaaaaaaaaa", new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :INVALID_PAYLOAD)

    # max acceptable number of keys (14)
    req = encrypted_request(dummy_payload(14), new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 200, :NONE)

    # no keys
    req = encrypted_request(dummy_payload(0), new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :NO_KEYS_IN_PAYLOAD)

    # too many keys
    req = encrypted_request(dummy_payload(15), new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, :TOO_MANY_KEYS)

    # rolling_period missing, too high, too low
    assert_tek_fails(:INVALID_ROLLING_PERIOD, rolling_period: 0)
    assert_tek_fails(:INVALID_ROLLING_PERIOD, rolling_period: 143)
    assert_tek_fails(:INVALID_ROLLING_PERIOD, rolling_period: 145)

    # risk level too high, too low
    assert_tek_fails(:INVALID_TRANSMISSION_RISK_LEVEL, transmission_risk_level: 9)
    assert_tek_fails(:INVALID_TRANSMISSION_RISK_LEVEL, transmission_risk_level: -1)

    # key data absent, too long, too short
    assert_tek_fails(:INVALID_KEY_DATA, key_data: nil)
    assert_tek_fails(:INVALID_KEY_DATA, key_data: '1'*15)
    assert_tek_fails(:INVALID_KEY_DATA, key_data: '1'*17)

    # key data absent, too long, too short
    assert_tek_fails(:INVALID_ROLLING_START_INTERVAL_NUMBER, rolling_start_interval_number: 0)
  end

  def test_invalid_sequencing
    teks = 14.times.map { tek }
    resp = post_teks(teks)
    assert_result(resp, 200, :NONE)

    # non-consecutive is no longer invalid.
    resp = post_teks(teks[0..3] + teks[5..-1])
    assert_result(resp, 200, :NONE)

    # shifted off of midnight
    resp = post_teks(teks.map { |tek| tek.rolling_start_interval_number += 1; tek })
    assert_result(resp, 200, :NONE)

    # only one tek shifted off of midnight
    resp = post_teks(teks.map.with_index { |tek, index| tek.rolling_start_interval_number += 1 if index == 4; tek })
    assert_result(resp, 400, :INVALID_ROLLING_START_INTERVAL_NUMBER)
  end

  def test_invalid_timestamp
    resp = post_timestamp(Time.now)
    assert_result(resp, 200, :NONE)

    resp = post_timestamp(Time.now - 3595)
    assert_result(resp, 200, :NONE)

    resp = post_timestamp(Time.now - 3605)
    assert_result(resp, 400, :INVALID_TIMESTAMP)

    resp = post_timestamp(Time.now + 3595)
    assert_result(resp, 200, :NONE)

    resp = post_timestamp(Time.now + 3605)
    assert_result(resp, 400, :INVALID_TIMESTAMP)
  end

  def post_timestamp(ts)
    payload = Covidshield::Upload.new(timestamp: ts, keys: [tek]).to_proto
    req = encrypted_request(payload, new_valid_keyset)
    @sub_conn.post('/upload', req.to_proto)
  end

  def post_teks(teks)
    ts = Time.now
    payload = Covidshield::Upload.new(timestamp: ts, keys: teks).to_proto
    req = encrypted_request(payload, new_valid_keyset)
    @sub_conn.post('/upload', req.to_proto)
  end

  def test_key_limit
    keys = (1..50).map { |n| key_n(n) }

    offset = 0

    keyset = new_valid_keyset

    upload_at_offset = ->(offset) {
      payload = Covidshield::Upload.new(
        timestamp: Time.now, keys: keys[offset...(offset+14)]
      ).to_proto
      req = encrypted_request(payload, keyset)
      @sub_conn.post('/upload', req.to_proto)
    }

    # Day zero, plus 14 subsequent days
    15.times do |offset|
      resp = upload_at_offset.call(offset)
      assert_result(resp, 200, :NONE)
    end

    # 15th subsequent day
    resp = upload_at_offset.call(15)
    assert_result(resp, 400, :INVALID_KEYPAIR)
  end

  private

  def key_n(n)
    tek(key_data: n.chr * 16)
  end

  def assert_tek_fails(code, **kwargs)
    # risk level too low
    req = encrypted_request(tek_payload(tek(**kwargs)), new_valid_keyset)
    resp = @sub_conn.post('/upload', req.to_proto)
    assert_result(resp, 400, code)
  end

  def tek_payload(tek)
    Covidshield::Upload.new(timestamp: Time.now, keys: [tek]).to_proto
  end

  def dummy_payload(nkeys=1, timestamp: Time.now)
    Covidshield::Upload.new(timestamp: timestamp, keys: nkeys.times.map{tek}).to_proto
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

  def tek(key_data: '1' * 16, transmission_risk_level: 3, rolling_period: 144, rolling_start_interval_number: next_rsin)
    Covidshield::TemporaryExposureKey.new(
      key_data: key_data,
      transmission_risk_level: transmission_risk_level,
      rolling_period: rolling_period,
      rolling_start_interval_number: rolling_start_interval_number
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
