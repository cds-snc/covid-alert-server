'use strict';

/*
*
*   Title : Metrics collection for Covid Alert
*   Purpose : Receive a json file from web or mobile app.
*   Author : Timothy Patrick Jodoin
*   Title : Principal Solution Architect
*   Email : tj@fsdcsolutions.com
*   Firm : FSDC Solutions Inc.
*   Date : Jan 12th, 2021
*   Client : CDS
*
*/

const
    AWS = require( 'aws-sdk' ),
    uuid = require( 'uuid' ),
    S3  = new AWS.S3(),
    fs = require( 'fs' );

exports.handler = async (event, context) => {

    let errorMessage = "Malformed Package";
    let transactionStatus = "FAILED";
    let bucket = process.env.dataBucket;           //encrypted env_var
    let fP = process.env.fileLoca;                  //encrypted env_var
    let d = new Date();
    let filename = uuid.v1();  // RFC4122 standard uuid Timestamp based generation


// informational for DEV
console.log("event", event);
console.log("event json : ", JSON.stringify(event));

	var body = JSON.stringify(event);

    // Is this an attack?
    if(body.includes("/script" || "< script >" || "alert(" || "SELECT " || "DROP " )){
        return Error(errorMessage);
    }

    //Screen out direct service attacks. Sanitize inputs
    else{

        console.log("bucket info :", bucket + "/" + fP);
        console.log("file name :", filename);

        //Probably a valid request do lets put it in the bucket.
        await S3.putObject({
            Bucket: bucket + "/" + fP,
            Key: filename + '.json',
            Body: body,
            ServerSideEncryption: 'aws:kms',        //Encrypt with KMS standard
            SSEKMSKeyId: process.env.dataKey,       //environment variable to encrypt - encryption key.
//            Metadata: config This is for immutable Metadata on S3

        })
        .promise()
        .then(() => {
            console.log('UPLOAD SUCCESS')           //Ya! We did it
            transactionStatus = "RECORD_CREATED";
        })
        .catch(e => {             //Oops, Check bucket policies and KMS key policies
            filename = "null";
            console.error('WRITE_ERROR - ', e);
            return Error("WRITE_ERROR", e);
         });


        //Build a receipt for positive response
        let response = {
            status: transactionStatus,
            time_stamp: d,
            Key: filename
        };

        //Return response
        return response;
    }
};
