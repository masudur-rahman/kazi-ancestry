#!/usr/bin/env python3
"""Local dev server for web/ that disables caching.

Python's default http.server lets browsers cache app.js/style.css, so edits
don't show up on reload. This sends no-store on everything.

Usage:  python3 tools/serve.py [port]   # default 8000, serves ./web
"""
import sys
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer


class NoCacheHandler(SimpleHTTPRequestHandler):
    def end_headers(self):
        self.send_header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
        self.send_header("Pragma", "no-cache")
        self.send_header("Expires", "0")
        super().end_headers()


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8000
    handler = partial(NoCacheHandler, directory="web")
    print(f"Kazi Ancestry → http://localhost:{port}  (no-cache; serving ./web)")
    ThreadingHTTPServer(("", port), handler).serve_forever()
