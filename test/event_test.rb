# frozen_string_literal: true

require_relative('lib/helper')
require('json')

class EventTest < MiniTest::Test
  include(Helper::Include)

  def test_generate_nonce

    # invalid method
    %w[get patch delete put].each do |meth|
      resp = @sub_conn.send(meth, '/event/nonce')
      assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")
    end

    # creates a nonce 
    resp = @sub_conn.post do |req|
      req.url('/event/nonce')
    end
    assert_response(resp, 200, 'text/plain; charset=utf-8')
    assert_match(/\A(.){32}\n\z/m, resp.body)

  end
end
