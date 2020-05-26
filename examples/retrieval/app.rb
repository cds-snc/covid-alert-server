# frozen_string_literal: true

require('faraday')
require('date')
require('openssl')
require_relative('../../test/lib/protocol/covidshield_pb')

class Database
  Fetch = Struct.new(:period)

  def initialize
    @fetches = []
  end

  def drop_old_data
    min_date = today_utc.prev_day(14)
    @fetches.reject! { |fetch| fetch.date < min_date }
  end

  def fetched?(period)
    @fetches.include?(Fetch.new(period))
  end

  def mark_fetched(period)
    return if @fetches.include?(Fetch.new(period))
    @fetches << Fetch.new(period)
  end

  private

  def today_utc
    Date.parse(Time.now.utc.strftime("%Y-%m-%d"))
  end
end

class App
  KEY_RETRIEVAL_URL = "http://127.0.0.1:8001"

  def initialize
    @database = Database.new
  end

  def run
    maybe_fetch_new_keys
    loop do
      sleep(60) # really this would more likely be an event loop
      # Realize that if you want to fetch *right* at the top of the hour, and
      # only then, you might miss if your clock is actually a couple of seconds
      # early -- so retrying 404, or scheduling past the top of the hour, or
      # really just checking as commonly as every couple of minutes whether
      # there's anything to fetch, rather than trying to run at the exact time
      # data becomes available, is probably correct.
      maybe_fetch_new_keys
    end
  end

  private

  def maybe_fetch_new_keys
    puts "maybe fetching new keys"
    hour = hour_number
    return if @last_successful_fetch == hour

    fetch_new_keys

    # we should probably check for exposure on a schedule that makes
    # more sense than after every key fetch
    check_for_exposure

    @last_successful_fetch = hour
  end

  def check_for_exposure
    puts "checking for exposure"
    fetch_exposure_config('302')
    puts "running exposure checks"
  end

  def fetch_new_keys
    puts "fetching new keys"

    @database.drop_old_data

    curr = current_period
    168.times do |n|
      period = curr - (2 * (n + 1))
      fetch_period(period) unless @database.fetched?(period)
    end
  end

  def fetch_exposure_config(region)
    puts "Fetching config: #{region}"
    resp = Faraday.get(exposure_configuration_url(region))
    raise("failed") unless resp.status == 200
    raise("failed") unless resp['Content-Type'] == 'application/json'
  end

  def fetch_period(period)
    puts "Fetching period: #{period}"
    resp = Faraday.get(period_url(period))
    raise("failed") unless resp.status == 200
    raise("failed") unless resp['Content-Type'] == 'application/zip'
    db_transaction do
      keys = send_to_framework(resp)
      puts("retrieved pack")
      @database.mark_fetched(period)
    end
  end

  def current_period
    (Time.now.to_i / 3600 / 2) * 2
  end

  def send_to_framework(resp)
    # See retrieve_test.rb for a ruby example of how to load this format, but probably,
    # you'll just be feeding the response body to the EN framework.
  end

  def db_transaction
    yield # not implemented here
  end

  def exposure_configuration_url(region)
    "#{KEY_RETRIEVAL_URL}/exposure-configuration/#{region}.json"
  end

  def period_url(period)
    message = "#{period}:#{hour_number}"
    key = [ENV.fetch("RETRIEVE_HMAC_KEY")].pack("H*")
    hmac = OpenSSL::HMAC.hexdigest("SHA256", key, message)
    "#{KEY_RETRIEVAL_URL}/retrieve/#{period}/#{hmac}"
  end

  def hour_number(at = Time.now)
    at.to_i / 3600
  end

  def today_utc
    Date.parse(Time.now.utc.strftime("%Y-%m-%d"))
  end
end

App.new.run if $PROGRAM_NAME == __FILE__
