# -*- coding: utf-8 -*-
# admin API 客户端(X-API-Key 头;响应信封 {code,msg,data},code=0 为成功)。
import json
import urllib.error
import urllib.parse
import urllib.request


class ApiError(Exception):
    def __init__(self, method, path, code, msg):
        Exception.__init__(self, '%s %s failed: code=%s msg=%s' % (method, path, code, msg))
        self.code = code


def items_of(data):
    # 列表信封兼容:{items:[...]} 与裸数组两种形态。
    if isinstance(data, dict):
        return data.get('items', [])
    if isinstance(data, list):
        return data
    return []


class AdminClient:
    def __init__(self, base, api_key, timeout=60):
        self.base = base.rstrip('/')
        self.key = api_key
        self.timeout = timeout

    def request(self, method, path, payload=None, params=None):
        url = self.base + path
        if params:
            url += '?' + urllib.parse.urlencode(params)
        headers = {'X-API-Key': self.key, 'Accept': 'application/json'}
        data = None
        if payload is not None:
            data = json.dumps(payload).encode('utf-8')
            headers['Content-Type'] = 'application/json'
        req = urllib.request.Request(url, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                body = json.loads(resp.read().decode('utf-8'))
        except urllib.error.HTTPError as exc:
            detail = exc.read()[:300].decode('utf-8', 'replace')
            raise ApiError(method, path, exc.code, detail)
        if not isinstance(body, dict) or body.get('code') != 0:
            raise ApiError(method, path, body.get('code') if isinstance(body, dict) else 'bad-envelope', body)
        return body.get('data')