# frozen_string_literal: true

require_relative('lib/helper')

class NewOutbreakEventTest < MiniTest::Test
  include(Helper::Include)

  def test_qr_submission
    resp = @sub_conn.post('/qr/new-event')
    assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")

    %w[get patch delete put].each do |meth|
      resp = @sub_conn.send(meth, '/qr/new-event')
      assert_response(resp, 401, 'text/plain; charset=utf-8', body: "unauthorized\n")
    end

    # too much post data
    resp = @sub_conn.post do |req|
      req.url('/qr/new-event')
      req.headers['Authorization'] = 'Bearer first-very-long-token'
      req.body = 'a'*2000
    end
    assert_result(resp, 400, :UNKNOWN)

    # bad data
    resp = @sub_conn.post do |req|
      req.url('/qr/new-event')
      req.headers['Authorization'] = 'Bearer first-very-long-token'
      req.body = 'a'
    end
    assert_result(resp, 400, :UNKNOWN)

    # bad location ID
    resp = @sub_conn.post do |req|
      req.url('/qr/new-event')
      req.headers['Authorization'] = 'Bearer first-very-long-token'
      req.body = Covidshield::OutbreakEvent.new(start_time: Time.now, end_time: Time.now, location_id: "a").to_proto
    end
    assert_result(resp, 400, :INVALID_ID)

    # start_time is 0
    resp = @sub_conn.post do |req|
      req.url('/qr/new-event')
      req.headers['Authorization'] = 'Bearer first-very-long-token'
      req.body = Covidshield::OutbreakEvent.new(start_time: Time.at(0), end_time: Time.now, location_id: "ABCDEFGH", severity: 1).to_proto
    end
    assert_result(resp, 400, :MISSING_TIMESTAMP)

    # end_time is 0
    resp = @sub_conn.post do |req|
      req.url('/qr/new-event')
      req.headers['Authorization'] = 'Bearer first-very-long-token'
      req.body = Covidshield::OutbreakEvent.new(start_time: Time.now,  end_time: Time.at(0), location_id: "ABCDEFGH", severity: 1).to_proto
    end
    assert_result(resp, 400, :MISSING_TIMESTAMP)

    # start_time >= end_time
    resp = @sub_conn.post do |req|
      req.url('/qr/new-event')
      req.headers['Authorization'] = 'Bearer first-very-long-token'
      req.body = Covidshield::OutbreakEvent.new(start_time: Time.now,  end_time: Time.now, location_id: "ABCDEFGH", severity: 1).to_proto
    end
    assert_result(resp, 400, :PERIOD_INVALID)

    # valid response
    resp = @sub_conn.post do |req|
      req.url('/qr/new-event')
      req.headers['Authorization'] = 'Bearer first-very-long-token'
      req.body = Covidshield::OutbreakEvent.new(start_time: Time.now,  end_time: Time.now+1, location_id: "ABCDEFGH", severity: 1).to_proto
    end
    assert_result(resp, 200, :NONE)
  end

  def assert_result(resp, code, error)
    assert_response(resp, code, 'application/x-protobuf')
    assert_equal(
      Covidshield::OutbreakEventResponse.new(error: error),
      Covidshield::OutbreakEventResponse.decode(resp.body)
    )
  end
end