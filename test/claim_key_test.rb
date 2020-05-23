# frozen_string_literal: true

require_relative('lib/helper')

class ClaimKeyTest < MiniTest::Test
  include(Helper::Include)

  def test_claim_key
    # too much post data
    resp = @sub_conn.post('/claim-key', 'a'*500)
    assert_response(resp, 400, 'application/x-protobuf')
    assert_fields(resp, error: :UNKNOWN, server_public_key: nil, tries_remaining: 8)

    # incomprehensible post data
    resp = @sub_conn.post('/claim-key', 'a'*100)
    assert_response(resp, 400, 'application/x-protobuf')
    assert_fields(resp, error: :UNKNOWN, server_public_key: nil, tries_remaining: 8)

    # invalid OneTimeCode
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: '12341234',
      app_public_key: '00001111222233334444555566667777'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 401, 'application/x-protobuf')
    assert_fields(resp, error: :INVALID_ONE_TIME_CODE, server_public_key: nil, tries_remaining: 7, remaining_ban_duration: 0)

    # app_public_key too short
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: new_valid_one_time_code,
      app_public_key: '0000111122223333'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 400, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_fields(resp, error: :INVALID_KEY, server_public_key: nil, tries_remaining: 7)

    # app_public_key too short
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: new_valid_one_time_code,
      app_public_key: '0000111122223333'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 400, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_fields(resp, error: :INVALID_KEY, server_public_key: nil, tries_remaining: 7)

    # app_public_key too long
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: new_valid_one_time_code,
      app_public_key: '000011112222333344445555666677778888'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 400, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_fields(resp, error: :INVALID_KEY, server_public_key: nil, tries_remaining: 7)

    # happy path (this resets the tries_remaining)
    5.times do |i|
      kcq = Covidshield::KeyClaimRequest.new(
        one_time_code: new_valid_one_time_code,
        app_public_key: "0000111122223333444455556666770#{i}"
      )
      resp = @sub_conn.post('/claim-key', kcq.to_proto)
      assert_response(resp, 200, 'application/x-protobuf')
      kcr = Covidshield::KeyClaimResponse.decode(resp.body)
      assert_equal(:NONE, kcr.error)
      assert_equal(32, kcr.server_public_key.each_byte.size)
      assert_equal(8, kcr.tries_remaining)
    end

    # app_public_key already exists
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: new_valid_one_time_code,
      app_public_key: '00001111222233334444555566667701'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 401, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(:INVALID_KEY, kcr.error)

    7.times do |i|
      resp = post_invalid_otc!
      assert_response(resp, 401, 'application/x-protobuf')
      assert_fields(resp, error: :INVALID_ONE_TIME_CODE, server_public_key: nil, tries_remaining: 8 - (1 + i), remaining_ban_duration: 0)
    end
    # Now, let's get banned
    resp = post_invalid_otc!
    assert_response(resp, 401, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(:INVALID_ONE_TIME_CODE, kcr.error)
    assert_equal(0, kcr.tries_remaining)
    assert_in_delta(3600, 5, kcr.remaining_ban_duration.seconds)

    # Now demonstrate that we're banned, even with a valid code
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: new_valid_one_time_code,
      app_public_key: "000011112222333344445555666677aa"
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 429, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(:TEMPORARY_BAN, kcr.error)
    assert_equal(0, kcr.tries_remaining)
    assert_in_delta(3600, 5, kcr.remaining_ban_duration.seconds)

    # Now expire the ban
    @dbconn.prepare(<<~SQL).execute
      UPDATE failed_key_claim_attempts SET last_failure = last_failure - INTERVAL 3600 SECOND
    SQL

    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: new_valid_one_time_code,
      app_public_key: "000011112222333344445555666677bb"
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 200, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(:NONE, kcr.error)
    assert_equal(32, kcr.server_public_key.each_byte.size)
    assert_equal(8, kcr.tries_remaining)
  end

  def post_invalid_otc!
    kcq = Covidshield::KeyClaimRequest.new(one_time_code: '12341234', app_public_key: '00001111222233334444555566667777')
    @sub_conn.post('/claim-key', kcq.to_proto)
  end

  def assert_fields(resp, fields)
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(Covidshield::KeyClaimResponse.new(**fields).to_json, kcr.to_json)
  end
end
