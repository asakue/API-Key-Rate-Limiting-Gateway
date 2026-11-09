"use client";

import { useEffect, useState } from "react";
import Navbar from "@/components/Navbar";
import { apiRequest, GATEWAY_URL } from "@/lib/api";
import {
  Terminal,
  Play,
  Zap,
  Clock,
  ArrowRight,
  ShieldCheck,
  AlertCircle,
  CheckCircle2,
} from "lucide-react";

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
}

export default function PlaygroundPage() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [apiKey, setApiKey] = useState("");
  const [method, setMethod] = useState("GET");
  const [path, setPath] = useState("/mock/api/v1/users");
  const [body, setBody] = useState("");
  const [loading, setLoading] = useState(false);

  // Response state
  const [responseStatus, setResponseStatus] = useState<number | null>(null);
  const [responseHeaders, setResponseHeaders] = useState<Record<string, string>>({});
  const [responseBody, setResponseBody] = useState<string>("");
  const [latencyMs, setLatencyMs] = useState<number | null>(null);

  // Burst test state
  const [burstResults, setBurstResults] = useState<{ id: number; status: number; remaining: string }[]>([]);
  const [bursting, setBursting] = useState(false);

  useEffect(() => {
    apiRequest<APIKey[]>("/api/keys")
      .then((data) => setKeys(data))
      .catch((err) => console.error(err));
  }, []);

  const handleSend = async () => {
    setLoading(true);
    setResponseStatus(null);
    setResponseHeaders({});
    setResponseBody("");
    setLatencyMs(null);

    const start = performance.now();

    try {
      const url = `${GATEWAY_URL}${path.startsWith("/") ? path : "/" + path}`;
      const headers: Record<string, string> = {};

      if (apiKey.trim()) {
        headers["Authorization"] = `Bearer ${apiKey.trim()}`;
      }

      const options: RequestInit = {
        method,
        headers,
      };

      if (method === "POST" && body.trim()) {
        headers["Content-Type"] = "application/json";
        options.body = body;
      }

      const res = await fetch(url, options);
      const elapsed = Math.round(performance.now() - start);

      const headerMap: Record<string, string> = {};
      res.headers.forEach((val, key) => {
        if (key.toLowerCase().startsWith("x-") || key.toLowerCase().includes("limit") || key.toLowerCase() === "content-type") {
          headerMap[key] = val;
        }
      });

      const text = await res.text();
      let formatted = text;
      try {
        formatted = JSON.stringify(JSON.parse(text), null, 2);
      } catch {}

      setResponseStatus(res.status);
      setResponseHeaders(headerMap);
      setResponseBody(formatted);
      setLatencyMs(elapsed);
    } catch (err: any) {
      setResponseStatus(0);
      setResponseBody(`Error contacting Gateway: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleBurstTest = async () => {
    if (!apiKey.trim()) {
      alert("Please provide an API key for the burst test!");
      return;
    }

    setBursting(true);
    setBurstResults([]);
    const count = 15;
    const url = `${GATEWAY_URL}${path.startsWith("/") ? path : "/" + path}`;

    const requests = Array.from({ length: count }, async (_, i) => {
      try {
        const res = await fetch(url, {
          method: "GET",
          headers: { Authorization: `Bearer ${apiKey.trim()}` },
        });
        const rem = res.headers.get("x-ratelimit-remaining") || "0";
        return { id: i + 1, status: res.status, remaining: rem };
      } catch (err: any) {
        return { id: i + 1, status: 0, remaining: "err" };
      }
    });

    const results = await Promise.all(requests);
    setBurstResults(results);
    setBursting(false);
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <Navbar />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center space-x-2">
            <Terminal className="w-6 h-6 text-blue-400" />
            <span>Interactive API Playground</span>
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Send live HTTP calls through the Gateway reverse proxy and observe rate limit headers in real-time
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* Left Column: Request Builder */}
          <div className="lg:col-span-5 bg-slate-900/60 border border-slate-800 rounded-xl p-6 space-y-4 shadow-xl">
            <h2 className="text-sm font-semibold text-white">Request Parameters</h2>

            {/* API Key Selector */}
            <div>
              <div className="flex justify-between items-center mb-1.5">
                <label className="text-xs font-medium text-slate-300">
                  API Key (gw_live_...)
                </label>
                {keys.length > 0 && (
                  <span className="text-[11px] text-slate-500">
                    {keys.length} key(s) registered
                  </span>
                )}
              </div>
              <input
                type="text"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="gw_live_paste_key_here..."
                className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 font-mono text-xs focus:outline-none focus:border-blue-500"
              />
            </div>

            {/* HTTP Method & Path */}
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1.5">
                Target Endpoint
              </label>
              <div className="flex space-x-2">
                <select
                  value={method}
                  onChange={(e) => setMethod(e.target.value)}
                  className="w-24 px-3 py-2 rounded-lg bg-slate-950 border border-slate-800 text-blue-400 text-xs font-bold focus:outline-none focus:border-blue-500"
                >
                  <option value="GET">GET</option>
                  <option value="POST">POST</option>
                </select>
                <input
                  type="text"
                  value={path}
                  onChange={(e) => setPath(e.target.value)}
                  placeholder="/mock/api/v1/users"
                  className="flex-1 px-3.5 py-2 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 font-mono text-xs focus:outline-none focus:border-blue-500"
                />
              </div>
              <div className="flex gap-2 mt-2">
                <button
                  type="button"
                  onClick={() => setPath("/mock/api/v1/users")}
                  className="text-[11px] px-2 py-0.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 font-mono"
                >
                  /mock/api/v1/users
                </button>
                <button
                  type="button"
                  onClick={() => setPath("/mock/api/v1/orders")}
                  className="text-[11px] px-2 py-0.5 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 font-mono"
                >
                  /mock/api/v1/orders
                </button>
              </div>
            </div>

            {/* Optional Request Body */}
            {method === "POST" && (
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">
                  Request JSON Body
                </label>
                <textarea
                  rows={3}
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  placeholder='{"name": "test payload"}'
                  className="w-full px-3.5 py-2 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 font-mono text-xs focus:outline-none focus:border-blue-500"
                />
              </div>
            )}

            {/* Action Buttons */}
            <div className="pt-2 flex flex-col sm:flex-row gap-3">
              <button
                onClick={handleSend}
                disabled={loading || bursting}
                className="flex-1 py-2.5 px-4 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white font-medium text-xs flex items-center justify-center space-x-2 shadow-lg shadow-blue-600/20 transition-all"
              >
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>{loading ? "Sending..." : "Send Request"}</span>
              </button>

              <button
                onClick={handleBurstTest}
                disabled={loading || bursting}
                className="py-2.5 px-4 rounded-lg bg-amber-600/20 border border-amber-500/30 hover:bg-amber-600/30 text-amber-300 font-medium text-xs flex items-center justify-center space-x-2 transition-all"
                title="Send 15 requests in parallel to test rate limiter"
              >
                <Zap className="w-3.5 h-3.5" />
                <span>{bursting ? "Testing..." : "Burst 15x (Test 429)"}</span>
              </button>
            </div>

            {/* Burst Results Preview */}
            {burstResults.length > 0 && (
              <div className="mt-4 p-3 rounded-lg bg-slate-950 border border-slate-800">
                <span className="text-xs font-medium text-slate-300 block mb-2">
                  Burst Rate Limit Test Results:
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {burstResults.map((b) => (
                    <span
                      key={b.id}
                      className={`text-[10px] px-1.5 py-0.5 rounded font-mono font-bold ${
                        b.status === 200
                          ? "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30"
                          : b.status === 429
                          ? "bg-amber-500/20 text-amber-400 border border-amber-500/30"
                          : "bg-rose-500/20 text-rose-400 border border-rose-500/30"
                      }`}
                    >
                      #{b.id}: {b.status}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Right Column: Live Response & Headers */}
          <div className="lg:col-span-7 bg-slate-900/60 border border-slate-800 rounded-xl p-6 space-y-4 shadow-xl flex flex-col">
            <div className="flex items-center justify-between pb-3 border-b border-slate-800">
              <h2 className="text-sm font-semibold text-white">Gateway Response</h2>
              {responseStatus !== null && (
                <div className="flex items-center space-x-3">
                  {latencyMs !== null && (
                    <span className="flex items-center space-x-1 text-xs text-slate-400">
                      <Clock className="w-3.5 h-3.5 text-cyan-400" />
                      <span>{latencyMs} ms</span>
                    </span>
                  )}
                  <span
                    className={`px-2.5 py-0.5 rounded text-xs font-bold font-mono ${
                      responseStatus === 200
                        ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/30"
                        : responseStatus === 429
                        ? "bg-amber-500/10 text-amber-400 border border-amber-500/30"
                        : "bg-rose-500/10 text-rose-400 border border-rose-500/30"
                    }`}
                  >
                    HTTP {responseStatus}
                  </span>
                </div>
              )}
            </div>

            {/* Injected Gateway Headers */}
            {Object.keys(responseHeaders).length > 0 && (
              <div>
                <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider block mb-2">
                  Injected Headers
                </span>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs font-mono">
                  {Object.entries(responseHeaders).map(([k, v]) => (
                    <div
                      key={k}
                      className="p-2 rounded bg-slate-950 border border-slate-800 flex justify-between"
                    >
                      <span className="text-slate-400 truncate mr-2">{k}:</span>
                      <span className="text-blue-400 font-bold truncate">{v}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Response Body */}
            <div className="flex-1 flex flex-col">
              <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider block mb-2">
                Response Payload
              </span>
              <pre className="flex-1 min-h-[220px] p-4 rounded-lg bg-slate-950 border border-slate-800 font-mono text-xs text-slate-300 overflow-auto whitespace-pre-wrap">
                {responseBody || (
                  <span className="text-slate-600">
                    Send a request or run a burst test to inspect the gateway output...
                  </span>
                )}
              </pre>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

