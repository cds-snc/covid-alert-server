<?php
  function generate_one_time_code() {
    $server_url = 'http://127.0.0.1:8000/new-key-claim';
    $options = array(
      'http' => array(
        'header' => 'Authorization: Bearer ' . $_ENV['KEY_CLAIM_TOKEN'],
        'method' => 'POST',
      )
    );

    $context = stream_context_create($options);
    $result = file_get_contents($server_url, false, $context);

    if ($result === FALSE) {
      throw new Exception('failed to issue code'); 
    }

    return $result;
  }

  echo generate_one_time_code();
?>
