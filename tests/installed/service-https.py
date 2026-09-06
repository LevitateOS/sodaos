#!/usr/bin/env python3
"""P06: configured-origin TLS/listener observation from the intended client.
No insecure mode, redirect journey, login, cookie jar or response-body capture.
"""
import argparse
from pathlib import Path
import ssl
import urllib.error
import urllib.parse
import urllib.request


def origin(value):
    p = urllib.parse.urlsplit(value)
    try:
        port = p.port
    except ValueError:
        raise argparse.ArgumentTypeError('invalid HTTPS origin') from None
    if p.scheme != 'https' or not p.hostname or p.username is not None or p.password is not None or p.path not in ('', '/') or '?' in value or '#' in value or port == 0:
        raise argparse.ArgumentTypeError('plain configured HTTPS origin required')
    return value.rstrip('/') + '/'


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def check(url, ca):
    p = Path(ca)
    if not p.is_absolute() or p.is_symlink() or not p.is_file() or p.stat().st_mode & 0o022:
        raise ValueError('absolute trusted regular CA file required')
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect(), urllib.request.HTTPSHandler(context=ssl.create_default_context(cafile=str(p))))
    try:
        response = opener.open(url, timeout=15)
    except urllib.error.HTTPError as response_error:
        if not 300 <= response_error.code < 400:
            raise
        response = response_error
    with response:
        if not 200 <= response.status < 400:
            raise ValueError('unexpected service response')
        print('Configured-origin TLS verified; HTTP status', response.status, '(not an authentication/product observation).')


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('origin', type=origin)
    p.add_argument('ca')
    a = p.parse_args()
    try:
        check(a.origin, a.ca)
    except Exception as err:
        # URLs, redirect queries and response bodies are not diagnostics.
        p.exit(1, 'HTTPS substrate check failed (' + type(err).__name__ + ').\n')
