# frozen_string_literal: true

require_relative('lib/helper')
require('rbnacl')

class EventTest < MiniTest::Test
  include(Helper::Include)

  def test_event_test
    # too much post data
    resp = @sub_conn.post('/event', 'a'*2000)
    assert_result(resp, 400, :INVALID_KEYS)

    # incomprehensible post data
    resp = @sub_conn.post('/event', 'a'*100)
    assert_result(resp, 400, :INVALID_KEYS)

    # happy path
    req = event_request("foo", new_valid_keyset)
    resp = @sub_conn.post('/event', req.to_proto)
    assert_result(resp, 200, :NONE)

    # app_public too long
    req = event_request("foo", new_valid_keyset, app_public_to_send:("a"*50))
    resp = @sub_conn.post('/event', req.to_proto)
    assert_result(resp, 400, :INVALID_KEYS)

    # app_public too short
    req = event_request("foo", new_valid_keyset, app_public_to_send:("a"*4))
    resp = @sub_conn.post('/event', req.to_proto)
    assert_result(resp, 400, :INVALID_KEYS)

    # server_public too short
    req = event_request("foo", new_valid_keyset, server_public_to_send:("a"*4))
    resp = @sub_conn.post('/event', req.to_proto)
    assert_result(resp, 400, :INVALID_KEYS)

    # server_public too long
    req = event_request("foo", new_valid_keyset, server_public_to_send:("a"*50))
    resp = @sub_conn.post('/event', req.to_proto)
    assert_result(resp, 400, :INVALID_KEYS)

    # server_public doesn't resolve to a server_private
    req = event_request("foo", new_valid_keyset, server_public_to_send:("a"*32))
    resp = @sub_conn.post('/event', req.to_proto)
    assert_result(resp, 401, :INVALID_KEYS)

    # server_public too short
    req = event_request("foo", new_valid_keyset, server_public_to_send:("a"*4))
    resp = @sub_conn.post('/event', req.to_proto)
    assert_result(resp, 400, :INVALID_KEYS)

    # server_public too long
    req = event_request("foo", new_valid_keyset, server_public_to_send:("a"*50))
    resp = @sub_conn.post('/event', req.to_proto)
    assert_result(resp, 400, :INVALID_KEYS)

  end

  def event_request(
    event, keyset, server_public: keyset[:server_public],
    app_public: keyset[:app_public], app_public_to_send: app_public,
    server_public_to_send: server_public
  )
    Covidshield::EventRequest.new(
      server_public_key: server_public_to_send.to_s,
      app_public_key: app_public_to_send.to_s,
      event: event,
    )
  end

  def assert_result(resp, code, error)
    assert_response(resp, code, 'application/x-protobuf')
    assert_equal(
      Covidshield::EventResponse.new(error: error),
      Covidshield::EventResponse.decode(resp.body)
    )
  end
end