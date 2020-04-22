# frozen_string_literal: true

require_relative('lib/helper')

class ClaimKeyTest < MiniTest::Test
  include(Helper::Include)

  def test_claim_key
    # too much post data
    resp = @sub_conn.post('/claim-key', 'a'*500)
    assert_response(resp, 400, 'application/x-protobuf')
    assert_fields(resp, error: :UNKNOWN, serverPublicKey: nil)

    # incomprehensible post data
    resp = @sub_conn.post('/claim-key', 'a'*100)
    assert_response(resp, 400, 'application/x-protobuf')
    assert_fields(resp, error: :UNKNOWN, serverPublicKey: nil)

    # invalid OneTimeCode
    kcq = Covidshield::KeyClaimRequest.new(
      oneTimeCode: '12341234',
      appPublicKey: '00001111222233334444555566667777'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 401, 'application/x-protobuf')
    assert_fields(resp, error: :INVALID_ONE_TIME_CODE, serverPublicKey: nil)

    # appPublicKey too short
    otc = new_valid_one_time_code
    kcq = Covidshield::KeyClaimRequest.new(
      oneTimeCode: new_valid_one_time_code,
      appPublicKey: '0000111122223333'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 400, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_fields(resp, error: :INVALID_KEY, serverPublicKey: nil)

    # appPublicKey too long
    otc = new_valid_one_time_code
    kcq = Covidshield::KeyClaimRequest.new(
      oneTimeCode: new_valid_one_time_code,
      appPublicKey: '000011112222333344445555666677778888'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 400, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_fields(resp, error: :INVALID_KEY, serverPublicKey: nil)

    # happy path
    5.times do |i|
      otc = new_valid_one_time_code
      kcq = Covidshield::KeyClaimRequest.new(
        oneTimeCode: new_valid_one_time_code,
        appPublicKey: "0000111122223333444455556666770#{i}"
      )
      resp = @sub_conn.post('/claim-key', kcq.to_proto)
      assert_response(resp, 200, 'application/x-protobuf')
      kcr = Covidshield::KeyClaimResponse.decode(resp.body)
      assert_equal(:NONE, kcr.error)
      assert_equal(32, kcr.serverPublicKey.each_byte.size)
    end

    # appPublicKey already exists
    otc = new_valid_one_time_code
    kcq = Covidshield::KeyClaimRequest.new(
      oneTimeCode: new_valid_one_time_code,
      appPublicKey: '00001111222233334444555566667701'
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 401, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(:INVALID_KEY, kcr.error)
  end

  def assert_fields(resp, fields)
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(Covidshield::KeyClaimResponse.new(**fields).to_json, kcr.to_json)
  end
end
