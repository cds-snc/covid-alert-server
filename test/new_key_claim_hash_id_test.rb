# frozen_string_literal: true

require_relative('lib/helper')

class NewKeyClaimhashIDTest < MiniTest::Test
  include(Helper::Include)

  def test_new_key_claim
    config = get_app_config()
    maxConsecutiveClaimKeyFailures = config["maxConsecutiveClaimKeyFailures"]
    
    resp = @sub_conn.post do |req|
      req.url('/new-key-claim/abcd')
      req.headers['Authorization'] = 'Bearer second-very-long-token'
    end
    assert_response(resp, 404, 'text/plain; charset=utf-8', body: "404 page not found\n")

    hash_id = random_hash

    %w[get patch delete put].each do |meth|
      resp = @sub_conn.send(meth, "/new-key-claim/#{hash_id}")
      assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")
    end

    resp = @sub_conn.post do |req|
      req.url("/new-key-claim/#{hash_id}")
      req.headers['Authorization'] = 'Bearer second-very-long-token'
    end
    assert_response(resp, 200, 'text/plain; charset=utf-8', body: /\A[0-9]{8}\n\z/m)
    previous_code = resp.body

    # Returns another code if hashID not claimed
    resp = @sub_conn.post do |req|
      req.url("/new-key-claim/#{hash_id}")
      req.headers['Authorization'] = 'Bearer second-very-long-token'
    end
    assert_response(resp, 200, 'text/plain; charset=utf-8', body: /\A[0-9]{8}\n\z/m)
    refute_equal(previous_code, resp.body)
    valid_code = resp.body

    # Ensure new codes are not generated for claimed hashIDs
    kcq = Covidshield::KeyClaimRequest.new(
      one_time_code: valid_code.strip,
      app_public_key: "00001111222233334444555566667710"
    )
    resp = @sub_conn.post('/claim-key', kcq.to_proto)
    assert_response(resp, 200, 'application/x-protobuf')
    kcr = Covidshield::KeyClaimResponse.decode(resp.body)
    assert_equal(:NONE, kcr.error)
    assert_equal(32, kcr.server_public_key.each_byte.size)
    assert_equal(maxConsecutiveClaimKeyFailures, kcr.tries_remaining)

    resp = @sub_conn.post do |req|
      req.url("/new-key-claim/#{hash_id}")
      req.headers['Authorization'] = 'Bearer second-very-long-token'
    end
    assert_response(resp, 403, 'text/plain; charset=utf-8', body: "forbidden\n")

  end
end