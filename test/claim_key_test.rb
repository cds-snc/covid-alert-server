# frozen_string_literal: true

require_relative('lib/helper')

class ClaimKeyTest < MiniTest::Test
  include(Helper::Include)

  def test_claim_key
    config = get_app_config()
    maxConsecutiveClaimKeyFailures = config["maxConsecutiveClaimKeyFailures"]
    
    # too much post data
    resp = @sub_conn.post('/claim-key', 'a'*500)
    assert_response(resp, 400, 'application/x-protobuf')
    assert_fields(resp, error: :UNKNOWN, server_public_key: nil, tries_remaining: maxConsecutiveClaimKeyFailures)

    # incomprehensible post data
    resp = @sub_conn.post('/claim-key', 'a'*100)
    assert_response(resp, 400, 'application/x-protobuf')
    assert_fields(resp, error: :UNKNOWN, server_public_key: nil, tries_remaining: maxConsecutiveClaimKeyFailures)

    # invalid OneTimeCode
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: '12341234',
      app_public_key: '00001111222233334444555566667777'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 401, 'application/x-protobuf')
    assert_fields(resp, error: :INVALID_ONE_TIME_CODE, server_public_key: nil, tries_remaining: maxConsecutiveClaimKeyFailures - 1, remaining_ban_duration: 0)

    # app_public_key too short
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: new_valid_one_time_code,
      app_public_key: '0000111122223333'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 400, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_fields(resp, error: :INVALID_KEY, server_public_key: nil, tries_remaining: maxConsecutiveClaimKeyFailures - 1)

    # app_public_key too short
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: new_valid_one_time_code,
      app_public_key: '0000111122223333'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 400, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_fields(resp, error: :INVALID_KEY, server_public_key: nil, tries_remaining: maxConsecutiveClaimKeyFailures - 1)

    # app_public_key too long
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: new_valid_one_time_code,
      app_public_key: '000011112222333344445555666677778888'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 400, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_fields(resp, error: :INVALID_KEY, server_public_key: nil, tries_remaining: maxConsecutiveClaimKeyFailures - 1)

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
      assert_equal(maxConsecutiveClaimKeyFailures, kcr.tries_remaining)
    end

    # Try code with spaces
    code_with_spaces = new_valid_one_time_code.insert(3, " ").insert(6, " ")
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: code_with_spaces,
      app_public_key: "00001111222233334444555566667706"
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 200, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(:NONE, kcr.error)
    assert_equal(32, kcr.server_public_key.each_byte.size)
    assert_equal(8, kcr.tries_remaining)

    # Try code with dashes
    code_with_dashes = new_valid_one_time_code.insert(3, "-").insert(6, "-")
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: code_with_dashes,
      app_public_key: "00001111222233334444555566667707"
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 200, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(:NONE, kcr.error)
    assert_equal(32, kcr.server_public_key.each_byte.size)
    assert_equal(8, kcr.tries_remaining)

    # app_public_key already exists
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: new_valid_one_time_code,
      app_public_key: '00001111222233334444555566667701'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 401, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(:INVALID_KEY, kcr.error)

    

    (maxConsecutiveClaimKeyFailures - 1).times do |i|
      resp = post_invalid_otc!
      assert_response(resp, 401, 'application/x-protobuf')
      assert_fields(resp, error: :INVALID_ONE_TIME_CODE, server_public_key: nil, tries_remaining: maxConsecutiveClaimKeyFailures - (1 + i), remaining_ban_duration: 0)
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
    assert_equal(maxConsecutiveClaimKeyFailures, kcr.tries_remaining)
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
