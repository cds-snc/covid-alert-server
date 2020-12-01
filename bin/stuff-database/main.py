import requests
import random
import string
import time
import multiprocessing as mp
from struct import *
from covidshield_pb2 import *
import nacl.utils
from nacl.public import PrivateKey, Box, PublicKey
import os

# Built for hash....

retrieve_hmac_key = os.environ['QA_HMAC']
ecdsa_key = os.environ['QA_ECDSA_KEY']
hcp_claim_token = os.environ['QA_BEARER_TOKEN']

# retrieve_hmac_key = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
# ecdsa_key = '30770201010420a6885a310b694b7bb4ba985459de1e79446dddcd1247c62ece925402b362a110a00a06082a8648ce3d030107a1440342000403eb64f714c4b4ed394331c26c31b7ce7156d00fb28982ad2679a87eaa1a3869802fbeb1d7ee28002762921929c3f7603672d535fcac3d24d57afbb4e2d97f5a'
# hcp_claim_token = 'thisisaverylongtoken'
# base_url = 'http://127.0.0.1'
# submit_port = ':8001'
# retrieve_port = ':8001'

region = '302'

app_priv_key = PrivateKey.generate()
app_key = bytes(app_priv_key.public_key)

start_key_num = 1

def gen_incrementing_tek():

    keys = []
    for i in range(14):
        en_id = int((time.time() - (i * 86400)) / (60 * 10))
        tek = TemporaryExposureKey()
        tek.key_data = nacl.utils.random(16)
        tek.transmission_risk_level = 4
        tek.rolling_start_interval_number = en_id
        tek.rolling_period = 144
        keys.append(tek) 
    return keys

# Submits a list of keys to the server
def submit(srv_pub, keys):
    global app_priv_key
    global app_key
    
    # url = base_url + submit_port + '/upload'
    url = 'https://submission.wild-samphire.cdssandbox.xyz/upload'

    headers = {'Content-Type': 'application/x-protobuf'}

    ep_req = EncryptedUploadRequest()
    ep_req.server_public_key = srv_pub
    ep_req.app_public_key = app_key
    
    nonce = bytes.fromhex('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa')

    upload = Upload()
    upload.timestamp.GetCurrentTime()
    for key in keys:
        k = upload.keys.add()
        k.key_data = key.key_data
        k.transmission_risk_level = key.transmission_risk_level
        k.rolling_start_interval_number = key.rolling_start_interval_number
        k.rolling_period = key.rolling_period

    upload_str = upload.SerializeToString()
    upload_box = Box(app_priv_key, PublicKey(srv_pub))

    ep_req.nonce = nonce
    ep_req.payload = upload_box.encrypt(upload_str, nonce)[Box.NONCE_SIZE:]

    resp = requests.post(url, headers=headers, data=ep_req.SerializeToString())

    ep_resp = EncryptedUploadResponse()
    ep_resp.ParseFromString(resp.content)

    print(ep_resp)

# Generate a new key claim token
def gen_key_claim_token() -> str:

    bearer_token = 'Bearer ' + hcp_claim_token
    headers = {'Authorization': bearer_token}

    # url = base_url + submit_port + '/new-key-claim'
    url = 'https://submission.wild-samphire.cdssandbox.xyz/new-key-claim'

    resp = requests.post(url, headers=headers)

    print('{}'.format(resp.text.strip()))

    return resp.text.strip()

# Claim a OTC for this app
def claim_key(otc) -> str:
    global app_key
    global app_priv_key

    url = 'https://submission.wild-samphire.cdssandbox.xyz/claim-key'
    # url = base_url + submit_port + '/claim-key'

    kc_req = KeyClaimRequest()
    kc_req.one_time_code = otc
    kc_req.app_public_key = app_key

    headers = {'Content-Type': 'application/x-protobuf'}
    
    resp = requests.post(url, headers=headers, data=kc_req.SerializeToString())

    kc_resp = KeyClaimResponse()
    kc_resp.ParseFromString(resp.content)

    return kc_resp.server_public_key

def stuff_key(): 
    otc = gen_key_claim_token()
    srv_pub = claim_key(otc)

    # # Submit 14 keys
    submit(srv_pub,gen_incrementing_tek())

if __name__ == '__main__':
    for i in range(1): 
        mp.Process(target=stuff_key).start()
