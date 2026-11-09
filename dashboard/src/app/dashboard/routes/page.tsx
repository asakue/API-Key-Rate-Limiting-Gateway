"use client";

import { useEffect, useState } from "react";
import Navbar from "@/components/Navbar";
import { apiRequest } from "@/lib/api";
import { Network, Plus, Trash2, X, Globe, ArrowRight, RefreshCw } from "lucide-react";

interface Route {
  id: string;
  name: string;
  path_prefix: string;
  upstream_url: string;
  strip_prefix: boolean;
  timeout_ms: number;
  is_active: boolean;
}

export default function RoutesPage() {
  const [routes, setRoutes] = useState<Route[]>([]);
  const [loading, setLoading] = useState(true);

  // Add Route Modal
  const [showAddModal, setShowAddModal] = useState(false);
  const [name, setName] = useState("");
  const [pathPrefix, setPathPrefix] = useState("");
  const [upstreamUrl, setUpstreamUrl] = useState("");
  const [stripPrefix, setStripPrefix] = useState(true);
  const [timeoutMs, setTimeoutMs] = useState(5000);
  const [saving, setSaving] = useState(false);

  const fetchRoutes = async () => {
    try {
      setLoading(true);
      const data = await apiRequest<Route[]>("/api/routes");
      setRoutes(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRoutes();
  }, []);

  const handleCreateRoute = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);

    try {
      await apiRequest("/api/routes", {
        method: "POST",
        body: JSON.stringify({
          name,
          path_prefix: pathPrefix,
          upstream_url: upstreamUrl,
          strip_prefix: stripPrefix,
          timeout_ms: Number(timeoutMs),
        }),
      });

      setShowAddModal(false);
      setName("");
      setPathPrefix("");
      setUpstreamUrl("");
      await fetchRoutes();
    } catch (err: any) {
      alert("Failed to add route: " + err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (routeId: string) => {
    if (!confirm("Are you sure you want to delete this upstream route?")) return;

    try {
      await apiRequest(`/api/routes/${routeId}`, { method: "DELETE" });
      await fetchRoutes();
    } catch (err: any) {
      alert("Failed to delete route: " + err.message);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <Navbar />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white flex items-center space-x-2">
              <Network className="w-6 h-6 text-blue-400" />
              <span>Upstream Services & Routing</span>
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Dynamically route incoming requests based on path prefix to your internal microservices
            </p>
          </div>

          <div className="flex items-center space-x-3">
            <button
              onClick={fetchRoutes}
              disabled={loading}
              className="p-2 rounded-lg bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200 transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin text-blue-400" : ""}`} />
            </button>
            <button
              onClick={() => setShowAddModal(true)}
              className="flex items-center space-x-1.5 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium shadow-lg shadow-blue-600/20 transition-all"
            >
              <Plus className="w-4 h-4" />
              <span>Add Upstream Route</span>
            </button>
          </div>
        </div>

        {/* Routes Table */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden shadow-xl">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-slate-900/90 text-xs font-semibold text-slate-400 border-b border-slate-800 uppercase tracking-wider">
                <tr>
                  <th className="py-3.5 px-4">Service Name</th>
                  <th className="py-3.5 px-4">Path Prefix</th>
                  <th className="py-3.5 px-4">Upstream Target URL</th>
                  <th className="py-3.5 px-4">Strip Prefix</th>
                  <th className="py-3.5 px-4">Timeout</th>
                  <th className="py-3.5 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60">
                {routes.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="py-12 text-center text-slate-500">
                      {loading ? "Loading routes..." : "No custom routes configured."}
                    </td>
                  </tr>
                ) : (
                  routes.map((r) => (
                    <tr key={r.id} className="hover:bg-slate-800/30 transition-colors">
                      <td className="py-3.5 px-4 font-medium text-slate-200">{r.name}</td>
                      <td className="py-3.5 px-4 font-mono text-xs text-blue-400">
                        <span className="bg-blue-500/10 px-2 py-1 rounded border border-blue-500/20">
                          {r.path_prefix}
                        </span>
                      </td>
                      <td className="py-3.5 px-4 font-mono text-xs text-slate-300">
                        <div className="flex items-center space-x-1.5">
                          <ArrowRight className="w-3.5 h-3.5 text-slate-500" />
                          <span>{r.upstream_url}</span>
                        </div>
                      </td>
                      <td className="py-3.5 px-4 text-xs">
                        {r.strip_prefix ? (
                          <span className="text-emerald-400">Yes (stripped)</span>
                        ) : (
                          <span className="text-slate-500">No (preserved)</span>
                        )}
                      </td>
                      <td className="py-3.5 px-4 font-mono text-xs text-slate-400">
                        {r.timeout_ms} ms
                      </td>
                      <td className="py-3.5 px-4 text-right">
                        <button
                          onClick={() => handleDelete(r.id)}
                          className="p-1.5 rounded text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 transition-colors"
                          title="Delete route"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Add Route Modal */}
        {showAddModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl">
              <div className="flex items-center justify-between mb-5">
                <h2 className="text-lg font-bold text-white">Add Upstream Route</h2>
                <button onClick={() => setShowAddModal(false)} className="text-slate-400 hover:text-white">
                  <X className="w-5 h-5" />
                </button>
              </div>

              <form onSubmit={handleCreateRoute} className="space-y-4">
                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1.5">
                    Service Name
                  </label>
                  <input
                    type="text"
                    required
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g. Orders Microservice"
                    className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1.5">
                    Path Prefix (e.g. /api/v1/orders)
                  </label>
                  <input
                    type="text"
                    required
                    value={pathPrefix}
                    onChange={(e) => setPathPrefix(e.target.value)}
                    placeholder="/api/v1/orders"
                    className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 text-sm font-mono focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1.5">
                    Upstream Target URL
                  </label>
                  <input
                    type="url"
                    required
                    value={upstreamUrl}
                    onChange={(e) => setUpstreamUrl(e.target.value)}
                    placeholder="http://orders-backend:8080"
                    className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 text-sm font-mono focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div className="flex items-center space-x-2 pt-1">
                  <input
                    type="checkbox"
                    id="strip"
                    checked={stripPrefix}
                    onChange={(e) => setStripPrefix(e.target.checked)}
                    className="rounded bg-slate-950 border-slate-800 text-blue-600 focus:ring-0"
                  />
                  <label htmlFor="strip" className="text-xs text-slate-300 cursor-pointer">
                    Strip prefix before forwarding to upstream
                  </label>
                </div>

                <div className="flex justify-end space-x-3 pt-3">
                  <button
                    type="button"
                    onClick={() => setShowAddModal(false)}
                    className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm font-medium"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={saving}
                    className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium shadow-lg shadow-blue-600/20 disabled:opacity-50"
                  >
                    {saving ? "Saving..." : "Add Route"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

