# -*-coding:utf-8-*-
import uuid

import requests

import json

import hmac

from hashlib import sha1

from hashlib import sha256

import urllib

import copy

import time

import curlify

# 填写您的accesskey 和secretKey

access_key = "15b62aa225ff4da7ad48e63c86c62adb"

secret_key = "8293c54bf23a4aeebe3fac307a7a928c"

# 签名计算


def sign(http_method, playlocd, servlet_path):

    time_str = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.localtime())

    playlocd["Timestamp"] = time_str

    parameters = copy.deepcopy(playlocd)

    parameters.pop("Signature")

    sorted_parameters = sorted(parameters.items(), key=lambda parameters: parameters[0])

    canonicalized_query_string = ""

    for (k, v) in sorted_parameters:
        canonicalized_query_string += "&" + percent_encode(k) + "=" + percent_encode(v)

    print("query string:\n", canonicalized_query_string[1:].encode("utf-8"))
    print("======================")

    string_to_sign = (
        http_method
        + "\n"
        + percent_encode(servlet_path)
        + "\n"
        + sha256(canonicalized_query_string[1:].encode("utf-8")).hexdigest()
    )
    print("string to sign", string_to_sign)
    print("======================\\\\\\\\\\\\xsjl")

    key = ("BC_SIGNATURE&" + secret_key).encode("utf-8")

    string_to_sign = string_to_sign.encode("utf-8")

    signature = hmac.new(key, string_to_sign, sha1).hexdigest()

    return signature


# 参数编码


def percent_encode(encode_str):

    encode_str = str(encode_str)

    res = urllib.parse.quote(encode_str.encode("utf-8"), "")

    res = res.replace("+", "%20")

    res = res.replace("*", "%2A")

    res = res.replace("%7E", "~")

    return res


if __name__ == "__main__":
    # http method

    method = "POST"

    # 目标域名 端口

    url = "https://%s" % ("api-beijing-2.cmecloud.cn:8443")

    # 请求url

    path = "/api/kcs/v2/clusters/f59ca7a8-fc17-4e82-aef2-4261a855f72f/nodes"

    # 可以不改

    headers = {"Content-Type": "application/json"}

    # 签名公参，如果有其他参数，同样在此添加

    querystring = {
        "AccessKey": access_key,
        "Timestamp": "2020-12-11T16:27:01Z",
        "Signature": "",
        "SignatureMethod": "HmacSHA1",
        "SignatureNonce": "",
        "SignatureVersion": "V2.0",
    }

    # 请求body

    payload = {
        "addType": "new",
        "infrastructure": {
            "flavor": "ecloud-normal-5118-1.0-000200040020",
            "serverType": "VM",
            "serverVmType": "common",
            "cpu": 2,
            "disk": 20,
            "ram": 4,
            "imageId": "Image For KCS_V7.5",
            "volumes": {
                "systemDisk": {"size": 50, "volumeType": "highPerformance"},
                "dataDisk": {"size": 50, "volumeType": "ebs_ceph_cache"},
            },
            "keypair": "hexintest",
            "SpecsName": "s1.large.2",
            "maxbandwidth": "1",
        },
    }

    # 生成SignatureNonce

    querystring["SignatureNonce"] = uuid.uuid4()

    # 生成签名

    querystring["Signature"] = sign(method, querystring, path)

    print(method, url + path, headers, querystring)

    test = requests.request(
        method, url + path, headers=headers, params=querystring, json=payload
    )

    # 转化为curl命令

    ci = curlify.to_curl(test.request)

    print("============================================================")

    # 将request转换成curl

    print(ci)

    print("============================================================")

    print("url : %s" % url)

    result = json.loads(test.text)

    print(result)

    try:
        assert result.get("state") == "OK"
        print("response: %s" % json.dumps(result, indent=4, ensure_ascii=False))
    except AssertionError as ae:
        print(json.dumps(result, indent=4, ensure_ascii=False))
