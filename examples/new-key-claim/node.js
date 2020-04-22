const http = require("http");

function generateOneTimeCode() {
  const url = new URL("http://127.0.0.1:8000/new-key-claim");

  const options = {
    hostname: url.hostname,
    port: url.port,
    path: url.pathname,
    protocol: url.protocol,
    method: "POST",
    headers: {
      Authorization: `Bearer ${process.env.KEY_CLAIM_TOKEN}`,
    },
  };

  const req = http.request(options, (res) => {
    res.on("data", (d) => {
      process.stdout.write(d);
    });
  });

  req.on("error", (error) => {
    console.error(error);
  });

  req.end();
}

generateOneTimeCode();
