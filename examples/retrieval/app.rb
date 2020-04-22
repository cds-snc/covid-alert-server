# frozen_string_literal: true

require('faraday')
require('date')
require('openssl')
require_relative('../../test/lib/protocol/covidshield_pb')

class Database
  Fetch = Struct.new(:date, :hour)

  def initialize
    @fetches = []
  end

  def drop_old_data
    min_date = today_utc.prev_day(14)
    @fetches.reject! { |fetch| fetch.date < min_date }
  end

  def has_all_hours?(date)
    (0..23).all? { |hour| has_hour?(date, hour) }
  end

  def has_hour?(date, hour)
    @fetches.include?(Fetch.new(date, hour))
  end

  def mark_hours_fetched(date)
    (0..23).each { |hour| mark_hour_fetched(date, hour) }
  end

  def mark_hour_fetched(date, hour)
    return if @fetches.include?(Fetch.new(date, hour))
    @fetches << Fetch.new(date, hour)
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
    fetch_exposure_config("ON")
    puts "running exposure checks"
  end

  def fetch_new_keys
    puts "fetching new keys"

    @database.drop_old_data

    (1..14).to_a.reverse.each do |days_ago| # [1, 14] => 14,13,...,2,1
      date = today_utc.prev_day(days_ago)
      fetch_date(date) unless @database.has_all_hours?(date)
    end

    # ... is exclusive range -- [0, hour)
    (0...current_hour_number_within_utc_day).each do |hour|
      date = today_utc
      fetch_hour(date, hour) unless @database.has_hour?(date, hour)
    end
  end

  def fetch_exposure_config(region)
    puts "Fetching config: #{region}"
    resp = Faraday.get(exposure_configuration_url(region))
    raise("failed") unless resp.status == 200
    raise("failed") unless resp['Content-Type'] == 'application/json'
  end

  def fetch_date(date)
    puts "Fetching date: #{date}"
    resp = Faraday.get(date_url(date))
    raise("failed") unless resp.status == 200
    raise("failed") unless resp['Content-Type'] == 'application/x-protobuf; delimited=true'
    db_transaction do
      keys = parse_and_save_keys_from(resp)
      puts("retrieved #{keys.size} keys")
      @database.mark_hours_fetched(date)
    end
  end

  def fetch_hour(date, hour)
    puts "Fetching hour: #{date} // #{hour}"
    resp = Faraday.get(hour_url(date, hour))
    raise("failed") unless resp.status == 200
    raise("failed") unless resp['Content-Type'] == 'application/x-protobuf; delimited=true'
    db_transaction do
      keys = parse_and_save_keys_from(resp)
      puts("retrieved #{keys.size} keys")
      @database.mark_hour_fetched(date, hour)
    end
  end

  BIG_ENDIAN_UINT32 = 'N'

  def parse_and_save_keys_from(resp)
    buf = resp.body.each_byte.to_a
    files = []
    until buf.empty?
      len = buf.shift(4).map(&:chr).join.unpack(BIG_ENDIAN_UINT32).first
      files << Covidshield::File.decode(buf.shift(len).map(&:chr).join)
    end
    files.flat_map(&:key)
  end

  def db_transaction
    yield # not implemented here
  end

  def exposure_configuration_url(region)
    "#{KEY_RETRIEVAL_URL}/exposure-configuration/#{region}.json"
  end

  def date_url(date)
    message = "#{date.iso8601}:#{format("%02d", hour_number)}"
    key = [ENV.fetch("RETRIEVE_HMAC_KEY")].pack("H*")
    hmac = OpenSSL::HMAC.hexdigest("SHA256", key, message)
    "#{KEY_RETRIEVAL_URL}/retrieve-day/#{date.iso8601}/#{hmac}"
  end

  def hour_url(date, hour)
    hour = format("%02d", hour)
    message = "#{date.iso8601}:#{hour}:#{format("%02d", hour_number)}"
    key = [ENV.fetch("RETRIEVE_HMAC_KEY")].pack("H*")
    hmac = OpenSSL::HMAC.hexdigest("SHA256", key, message)
    "#{KEY_RETRIEVAL_URL}/retrieve-hour/#{date.iso8601}/#{hour}/#{hmac}"
  end

  def current_hour_number_within_utc_day
    (Time.now.to_i % 86400) / 3600
  end

  def hour_number(at = Time.now)
    at.to_i / 3600
  end

  def today_utc
    Date.parse(Time.now.utc.strftime("%Y-%m-%d"))
  end
end

App.new.run if $PROGRAM_NAME == __FILE__
