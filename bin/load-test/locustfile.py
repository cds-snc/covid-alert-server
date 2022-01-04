import hashlib
import hmac
import math
import os
import time

from locust import HttpUser, task

HMAC_KEY = os.getenv("HMAC_KEY")

class KeyRetrieval(HttpUser):

    @task
    def get_teks_current(self):
        "Retrieve Temporary Exposure Keys for current period"
        result = self.get_sig(time.time())
        self.client.get(f"/retrieve/302/{result[0]}/{result[1]}")

    @task
    def get_teks_previous(self):
        "Retrieve Temporary Exposure Keys for previous period (1 hour before)"
        result = self.get_sig(time.time() - 3600)
        self.client.get(f"/retrieve/302/{result[0]}/{result[1]}")       

    @task
    def get_teks_future(self):
        "Retrieve Temporary Exposure Keys for future period (1 hour ahead)"
        result = self.get_sig(time.time() + 3600)
        self.client.get(f"/retrieve/302/{result[0]}/{result[1]}")

    def get_sig(self, timestamp):
        "Generates a signed message for the given timestamp"
        period = math.floor(timestamp / 86400)
        hour = math.floor(timestamp / 3600)
        message = f"302:{period}:{hour}"

        signature = hmac.new(
            bytes.fromhex(HMAC_KEY), 
            msg=bytes(message, "utf-8"), 
            digestmod=hashlib.sha256).hexdigest()

        return period, signature

