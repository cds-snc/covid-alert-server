'use strict';

const
    AWS = require('aws-sdk'),
    uuid = require('uuid'),
    S3 = new AWS.S3();

exports.handler = async (event, context) => {

    const bucket = process.env.dataBucket;
    const filePath = process.env.fileLoca;
    const kmsKey = process.env.dataKey;
    let filename = uuid.v1();
    let transactionStatus = "FAILED";
    var body = JSON.stringify(event);

    const bucketParams = {
        Bucket: bucket + "/" + filePath,
        Key: filename + '.json',
        Body: body,
        ServerSideEncryption: 'aws:kms',
        SSEKMSKeyId: kmsKey
    };

    try {
        let resp = await S3.putObject(bucketParams).promise();
        console.log(resp);
        transactionStatus = { "status": "RECORD_CREATED", "key": filename};
    } catch (err) {
        console.log("UPLOAD FAILED", err);
        transactionStatus = { "status": "UPLOAD FAILED"};
    }

    return transactionStatus;
};
