# frozen_string_literal: true

require_relative('lib/helper')

class NewKeyClaimTest < MiniTest::Test
  include(Helper::Include)

  def test_new_key_claim
    resp = @sub_conn.post('/new-key-claim')
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    resp = @sub_conn.post do |req|
      req.url('/new-key-claim')
      req.headers['Authorization'] = "Bearer not-the-right-token"
    end
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    resp = @sub_conn.post do |req|
      req.url('/new-key-claim')
      req.headers['Authorization'] = 'Bearer first-token'
    end
    assert_response(resp, 200, 'text/plain; charset=utf-8', body: /\A[0-9]{8}\n\z/m)

    resp = @sub_conn.post do |req|
      req.url('/new-key-claim')
      req.headers['Authorization'] = 'Bearer second-token'
    end
    assert_response(resp, 200, 'text/plain; charset=utf-8', body: /\A[0-9]{8}\n\z/m)

    %w[get patch delete put].each do |meth|
      resp = @sub_conn.send(meth, '/new-key-claim')
      assert_response(resp, 405, 'text/plain; charset=utf-8', body: "method not allowed\n")
    end
  end
end
