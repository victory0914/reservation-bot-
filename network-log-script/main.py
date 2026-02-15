from mitmproxy import http
import json

RECORDED = []

def response(flow: http.HTTPFlow):
    print("Response hook called for: ", flow.request.pretty_url)
    # Filter for Gurunavi domains (English & Japanese)
    if "cityheaven.net" not in flow.request.pretty_url and "gnavi.co.jp" not in flow.request.pretty_url:
        return

    # Skip static assets to keep log clean
    if flow.request.path.split('?')[0].lower().endswith(
        (".png", ".jpg", ".jpeg", ".gif", ".css", ".js", ".woff", ".woff2", ".ico", ".svg")
    ):
        return

    # Capture Request & Response Pair
    entry = {
        "method": flow.request.method,
        "url": flow.request.pretty_url,
        "request": {
            "headers": dict(flow.request.headers),
            "body": flow.request.get_text()[:5000] if flow.request.content else "",  # Truncate large bodies
        },
        "response": {
            "status": flow.response.status_code,
            "headers": dict(flow.response.headers),
            "body": flow.response.get_text()[:10000] if flow.response.content else "", # Capture enough HTML to find tokens
        }
    }
    
    RECORDED.append(entry)
    print(f"Captured: {flow.request.method} {flow.request.pretty_url}")

def done():
    filename = "city_heaven.json"
    with open(filename, "w", encoding="utf-8") as f:
        json.dump(RECORDED, f, indent=2, ensure_ascii=False)
    print(f"Saved {len(RECORDED)} interactions to {filename}")

