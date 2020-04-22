# frozen_string_literal: true

require_relative('lib/helper')
require('json')

class RetrieveTest < MiniTest::Test
  include(Helper::Include)

  def test_get_exposure_for_region
    # config doesn't exist
    resp = @ret_conn.send(:get, "/something/nonsense")
    assert_response(resp, 404, 'text/plain; charset=utf-8', body: "404 page not found\n")

    # region doesn't exist
    resp = get_exposure_config('non-region')
    assert_response(resp, 404, 'text/plain; charset=utf-8', body: "404 page not found\n")

    # valid regional data
    resp = get_exposure_config('ON')
    assert_response(resp, 200, 'application/json')
    assert(JSON.parse(resp.body).is_a?(Hash), 'response should be a valid JSON dictionary')
  end
end
