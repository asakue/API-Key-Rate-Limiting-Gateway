"use client";

import { useEffect, useState } from "react";
import Navbar from "@/components/Navbar";
import { apiRequest } from "@/lib/api";
import Link from "next/link";
import {
  Activity,
  CheckCircle2,
  Ban,
  AlertTriangle,
  Clock,
  KeyRound,
  ArrowUpRight,
  RefreshCw,
  Terminal,
} from "lucide-react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

interface MetricSummary {
  total_requests: number;
  total_2xx: number;
  total_4xx: number;
  total_5xx: number;
  total_429: number;
  average_latency_ms: number;
  active_keys_count: number;
}

interface TimeSeriesPoint {
  timestamp: string;
  total: number;
  success: number;
  client_err: number;
  throttled: number;
  server_err: number;
}

export default function DashboardOverviewPage() {
  const [summary, setSummary] = useState<MetricSummary | null>(null);
  const [timeSeries, setTimeSeries] = useState<TimeSeriesPoint[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchMetrics = async () => {
    try {
      setLoading(true);
      const [sumData, tsData] = await Promise.all([
        apiRequest<MetricSummary>("/api/metrics/summary"),
        apiRequest<TimeSeriesPoint[]>("/api/metrics/timeseries"),
      ]);
      setSummary(sumData);
      setTimeSeries(
        tsData.map((d) => ({
          ...d,
          timeLabel: new Date(d.timestamp).toLocaleTimeString([], {
            hour: "2-digit",
            minute: "2-digit",
          }),
        }))
      );
    } catch (err) {
      console.error("Failed to load metrics", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 5000);
    return () => clearInterval(interval);
  }, []);

  const successRate =
    summary && summary.total_requests > 0
      ? ((summary.total_2xx / summary.total_requests) * 100).toFixed(1)
      : "100.0";

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <Navbar />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
        {/* Header section */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white">System Overview</h1>
            <p className="text-sm text-slate-400 mt-1">
              Real-time reverse proxy telemetry, rate limiting blocks, and traffic metrics
            </p>
          </div>

          <div className="flex items-center space-x-3">
            <button
              onClick={fetchMetrics}
              disabled={loading}
              className="p-2 rounded-lg bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200 transition-colors"
              title="Refresh metrics"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin text-blue-400" : ""}`} />
            </button>
            <Link
              href="/dashboard/playground"
              className="flex items-center space-x-1.5 px-3.5 py-2 rounded-lg bg-slate-900 border border-slate-800 hover:border-slate-700 text-slate-200 text-xs font-medium transition-all"
            >
              <Terminal className="w-3.5 h-3.5 text-blue-400" />
              <span>Open Playground</span>
            </Link>
            <Link
              href="/dashboard/keys"
              className="flex items-center space-x-1.5 px-3.5 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium shadow-lg shadow-blue-600/20 transition-all"
            >
              <KeyRound className="w-3.5 h-3.5" />
              <span>Generate Key</span>
            </Link>
          </div>
        </div>

        {/* Metric Cards Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* Total Requests */}
          <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-5 hover:border-slate-700 transition-colors">
            <div className="flex items-center justify-between text-slate-400 mb-3">
              <span className="text-xs font-medium">Total Requests (24h)</span>
              <Activity className="w-4 h-4 text-blue-400" />
            </div>
            <div className="text-3xl font-extrabold text-white tracking-tight">
              {summary ? summary.total_requests.toLocaleString() : "0"}
            </div>
            <div className="text-xs text-slate-500 mt-2 flex items-center">
              <span>Proxied through Go reverse proxy</span>
            </div>
          </div>

          {/* Success Rate */}
          <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-5 hover:border-slate-700 transition-colors">
            <div className="flex items-center justify-between text-slate-400 mb-3">
              <span className="text-xs font-medium">Success Rate (2xx)</span>
              <CheckCircle2 className="w-4 h-4 text-emerald-400" />
            </div>
            <div className="text-3xl font-extrabold text-white tracking-tight">
              {successRate}%
            </div>
            <div className="text-xs text-slate-500 mt-2">
              {summary ? summary.total_2xx.toLocaleString() : 0} successful calls
            </div>
          </div>

          {/* Throttled (429) */}
          <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-5 hover:border-slate-700 transition-colors">
            <div className="flex items-center justify-between text-slate-400 mb-3">
              <span className="text-xs font-medium">Throttled (429 Blocked)</span>
              <Ban className="w-4 h-4 text-amber-400" />
            </div>
            <div className="text-3xl font-extrabold text-white tracking-tight">
              {summary ? summary.total_429.toLocaleString() : "0"}
            </div>
            <div className="text-xs text-amber-500/80 mt-2">
              Protected by Sliding Window Limiter
            </div>
          </div>

          {/* Average Latency */}
          <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-5 hover:border-slate-700 transition-colors">
            <div className="flex items-center justify-between text-slate-400 mb-3">
              <span className="text-xs font-medium">Data Plane Overhead</span>
              <Clock className="w-4 h-4 text-cyan-400" />
            </div>
            <div className="text-3xl font-extrabold text-white tracking-tight">
              {summary && summary.total_requests > 0
                ? `${summary.average_latency_ms.toFixed(1)} ms`
                : "< 1.5 ms"}
            </div>
            <div className="text-xs text-slate-500 mt-2 flex items-center">
              <span>Atomic Redis Lua execution</span>
            </div>
          </div>
        </div>

        {/* Activity Chart */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-6">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-base font-semibold text-white">Traffic Activity (Requests / Min)</h2>
              <p className="text-xs text-slate-400 mt-0.5">
                Breakdown by Success (2xx), Throttled (429), and Errors (4xx/5xx)
              </p>
            </div>
            <div className="flex items-center space-x-4 text-xs">
              <div className="flex items-center space-x-1.5">
                <span className="w-2.5 h-2.5 rounded-full bg-blue-500"></span>
                <span className="text-slate-300">2xx Success</span>
              </div>
              <div className="flex items-center space-x-1.5">
                <span className="w-2.5 h-2.5 rounded-full bg-amber-500"></span>
                <span className="text-slate-300">429 Throttled</span>
              </div>
              <div className="flex items-center space-x-1.5">
                <span className="w-2.5 h-2.5 rounded-full bg-rose-500"></span>
                <span className="text-slate-300">Errors (4xx/5xx)</span>
              </div>
            </div>
          </div>

          <div className="h-72 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={timeSeries}>
                <defs>
                  <linearGradient id="colorSuccess" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="colorThrottled" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="#f59e0b" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="colorErrors" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#ef4444" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="#ef4444" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
                <XAxis dataKey="timeLabel" stroke="#64748b" fontSize={11} />
                <YAxis stroke="#64748b" fontSize={11} allowDecimals={false} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "#0f172a",
                    borderColor: "#334155",
                    borderRadius: "8px",
                    color: "#f8fafc",
                    fontSize: "12px",
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="success"
                  name="2xx Success"
                  stroke="#3b82f6"
                  fillOpacity={1}
                  fill="url(#colorSuccess)"
                />
                <Area
                  type="monotone"
                  dataKey="throttled"
                  name="429 Throttled"
                  stroke="#f59e0b"
                  fillOpacity={1}
                  fill="url(#colorThrottled)"
                />
                <Area
                  type="monotone"
                  dataKey="client_err"
                  name="4xx Errors"
                  stroke="#ef4444"
                  fillOpacity={1}
                  fill="url(#colorErrors)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Quick Start / Gateway Info */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-6">
            <h3 className="text-sm font-semibold text-white mb-2 flex items-center space-x-2">
              <span className="text-blue-400">#</span>
              <span>Client Integration Example</span>
            </h3>
            <p className="text-xs text-slate-400 mb-4">
              Pass your generated secret API key in the standard Bearer header:
            </p>
            <div className="bg-slate-950 rounded-lg p-3.5 border border-slate-800/80 font-mono text-xs text-slate-300 overflow-x-auto">
              <code>{`curl -i http://localhost:8000/mock/api/v1/users \\
  -H "Authorization: Bearer gw_live_your_key_here"`}</code>
            </div>
          </div>

          <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-6">
            <h3 className="text-sm font-semibold text-white mb-2 flex items-center space-x-2">
              <span className="text-emerald-400">#</span>
              <span>Rate Limit Response Headers</span>
            </h3>
            <p className="text-xs text-slate-400 mb-4">
              The gateway injects standard RFC rate limit headers into every client response:
            </p>
            <div className="space-y-2 text-xs font-mono">
              <div className="flex justify-between p-2 rounded bg-slate-950 border border-slate-800">
                <span className="text-slate-400">X-RateLimit-Limit:</span>
                <span className="text-blue-400">60</span>
              </div>
              <div className="flex justify-between p-2 rounded bg-slate-950 border border-slate-800">
                <span className="text-slate-400">X-RateLimit-Remaining:</span>
                <span className="text-emerald-400">42</span>
              </div>
              <div className="flex justify-between p-2 rounded bg-slate-950 border border-slate-800">
                <span className="text-slate-400">X-RateLimit-Reset:</span>
                <span className="text-amber-400">18 (seconds)</span>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

