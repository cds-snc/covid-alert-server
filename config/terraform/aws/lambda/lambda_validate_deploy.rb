require 'net/http'
require 'aws-sdk-codedeploy'

def handler(event:, context:)
  logger = Logger.new($stdout)
  uri = URI.parse(ENV['TEST_LB_ENDPOINT'])

  def codedeploy_status(status, event)
    client = Aws::CodeDeploy::Client.new
    client.put_lifecycle_event_hook_execution_status(
      deployment_id: event['DeploymentId'],
      lifecycle_event_hook_execution_id: event['LifecycleEventHookExecutionId'],
      status: status
    )
  end

  begin
    resp = Net::HTTP.start(
      uri.host,
      uri.port,
      use_ssl: true, open_timeout: 5,
      read_timeout: 5,
      verify_mode: OpenSSL::SSL::VERIFY_NONE
    ) do |http|
      request = Net::HTTP::Get.new uri
      request['covidshield'] = ENV['CLOUDFRONT_CUSTOM_HEADER'] if ENV['CLOUDFRONT_CUSTOM_HEADER']
      response = http.request request
    end
  rescue StandardError => error
    status = 'Failed'
    codedeploy_status(status, event)
    logger.info("Validation of test listener failed with error: #{error.message}")
  end

  if resp && resp.code == '200' && resp.body.start_with?('OK')
    status = 'Succeeded'
    codedeploy_status(status, event)
    logger.info("Validation of test listener status code #{status}: #{resp}")
  else
    status = 'Failed'
    codedeploy_status(status, event)
    logger.info("Validation of test listener status code #{status}: #{resp}")
  end
end
