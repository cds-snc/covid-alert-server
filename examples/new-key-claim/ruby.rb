require('net/http')
require('uri')

module OneTimeCode
  SERVER_URI = URI.parse('http://127.0.0.1:8000/new-key-claim')
  TOKEN = ENV.fetch('KEY_CLAIM_TOKEN')

  Error = Class.new(StandardError)

  def self.generate
    token = nil
    Net::HTTP.start(SERVER_URI.host, SERVER_URI.port, use_ssl: SERVER_URI.scheme == 'https') do |http|
      request = Net::HTTP::Post.new(SERVER_URI.path, 'Authorization' => "Bearer #{TOKEN}")
      response = http.request(request)
      unless response.code.to_i == 200
        raise(Error, "failed to issue code: (#{response.code}) #{response.body}")
      end
      token = response.body.chomp
    end
    token
  end
end

puts OneTimeCode.generate
