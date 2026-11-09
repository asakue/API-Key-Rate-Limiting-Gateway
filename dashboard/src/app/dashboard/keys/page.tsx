"use client";

import { useEffect, useState } from "react";
import Navbar from "@/components/Navbar";
import { apiRequest } from "@/lib/api";
import {
  KeyRound,
  Plus,
  Copy,
  Check,
  AlertTriangle,
  Trash2,
  ShieldAlert,
  X,
  RefreshCw,
} from "lucide-react";

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  tier_id: string;
  tier_name: string;
  scopes: string[];
  status: string;
  rate_limit_per_min: number;
  monthly_quota: number;
  created_at: string;
  revoked_at?: string;
}

interface Tier {
  id: string;
  name: string;
  rate_limit_per_min: number;
  monthly_quota: number;
}

export default function KeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [tiers, setTiers] = useState<Tier[]>([]);
  const [loading, setLoading] = useState(true);

  // Create Key Modal State
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [name, setName] = useState("");
  const [tierId, setTierId] = useState("tier_free");
  const [scopesStr, setScopesStr] = useState("*");
  const [creating, setCreating] = useState(false);

  // Reveal Modal State (shown ONCE)
  const [revealedKey, setRevealedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [keysData, tiersData] = await Promise.all([
        apiRequest<APIKey[]>("/api/keys"),
        apiRequest<Tier[]>("/api/tiers"),
      ]);
      setKeys(keysData);
      setTiers(tiersData);
      if (tiersData.length > 0 && !tierId) {
        setTierId(tiersData[0].id);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);

    try {
      const scopes = scopesStr
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);

      const res = await apiRequest<{ raw_key: string }>("/api/keys", {
        method: "POST",
        body: JSON.stringify({
          name,
          tier_id: tierId,
          scopes: scopes.length > 0 ? scopes : ["*"],
        }),
      });

      setShowCreateModal(false);
      setName("");
      setScopesStr("*");
      setRevealedKey(res.raw_key);
      await fetchData();
    } catch (err: any) {
      alert("Failed to create key: " + err.message);
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = async (keyId: string) => {
    if (!confirm("Are you sure you want to revoke this API key? Traffic using it will be blocked immediately.")) {
      return;
    }

    try {
      await apiRequest(`/api/keys/${keyId}`, { method: "DELETE" });
      await fetchData();
    } catch (err: any) {
      alert("Failed to revoke key: " + err.message);
    }
  };

  const handleCopy = () => {
    if (revealedKey) {
      navigator.clipboard.writeText(revealedKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <Navbar />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white flex items-center space-x-2">
              <KeyRound className="w-6 h-6 text-blue-400" />
              <span>API Keys Management</span>
            </h1>
            <p className="text-sm text-slate-400 mt-1">
              Issue prefix-structured keys, set endpoints scopes, and manage access limits
            </p>
          </div>

          <div className="flex items-center space-x-3">
            <button
              onClick={fetchData}
              disabled={loading}
              className="p-2 rounded-lg bg-slate-900 border border-slate-800 text-slate-400 hover:text-slate-200 transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin text-blue-400" : ""}`} />
            </button>
            <button
              onClick={() => setShowCreateModal(true)}
              className="flex items-center space-x-1.5 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium shadow-lg shadow-blue-600/20 transition-all"
            >
              <Plus className="w-4 h-4" />
              <span>Create New Key</span>
            </button>
          </div>
        </div>

        {/* Keys Table Card */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden shadow-xl">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-slate-900/90 text-xs font-semibold text-slate-400 border-b border-slate-800 uppercase tracking-wider">
                <tr>
                  <th className="py-3.5 px-4">Name</th>
                  <th className="py-3.5 px-4">Key Preview</th>
                  <th className="py-3.5 px-4">Tier & Quota</th>
                  <th className="py-3.5 px-4">Scopes</th>
                  <th className="py-3.5 px-4">Status</th>
                  <th className="py-3.5 px-4">Created</th>
                  <th className="py-3.5 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60">
                {keys.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="py-12 text-center text-slate-500">
                      {loading ? "Loading keys..." : "No API keys created yet. Click 'Create New Key' to generate one."}
                    </td>
                  </tr>
                ) : (
                  keys.map((k) => (
                    <tr key={k.id} className="hover:bg-slate-800/30 transition-colors">
                      <td className="py-3.5 px-4 font-medium text-slate-200">{k.name}</td>
                      <td className="py-3.5 px-4 font-mono text-xs text-slate-400">
                        <span className="bg-slate-950 px-2 py-1 rounded border border-slate-800">
                          {k.key_prefix}
                        </span>
                      </td>
                      <td className="py-3.5 px-4">
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-500/10 text-blue-400 border border-blue-500/20">
                          {k.tier_name || k.tier_id}
                        </span>
                        <span className="text-xs text-slate-500 ml-2">
                          ({k.rate_limit_per_min} RPM)
                        </span>
                      </td>
                      <td className="py-3.5 px-4 font-mono text-xs text-slate-400">
                        {k.scopes?.join(", ") || "*"}
                      </td>
                      <td className="py-3.5 px-4">
                        {k.status === "active" ? (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                            Active
                          </span>
                        ) : (
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-rose-500/10 text-rose-400 border border-rose-500/20">
                            Revoked
                          </span>
                        )}
                      </td>
                      <td className="py-3.5 px-4 text-xs text-slate-500">
                        {new Date(k.created_at).toLocaleDateString()}
                      </td>
                      <td className="py-3.5 px-4 text-right">
                        {k.status === "active" && (
                          <button
                            onClick={() => handleRevoke(k.id)}
                            className="p-1.5 rounded text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 transition-colors"
                            title="Revoke key immediately"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        )}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Modal: Create Key */}
        {showCreateModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl">
              <div className="flex items-center justify-between mb-5">
                <h2 className="text-lg font-bold text-white">Create New API Key</h2>
                <button
                  onClick={() => setShowCreateModal(false)}
                  className="text-slate-400 hover:text-white"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <form onSubmit={handleCreateKey} className="space-y-4">
                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1.5">
                    Key Name / Description
                  </label>
                  <input
                    type="text"
                    required
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g. Production iOS Client"
                    className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1.5">
                    Tier & Limits
                  </label>
                  <select
                    value={tierId}
                    onChange={(e) => setTierId(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-blue-500"
                  >
                    {tiers.map((t) => (
                      <option key={t.id} value={t.id}>
                        {t.name} ({t.rate_limit_per_min} req/min, {t.monthly_quota.toLocaleString()} / mo)
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1.5">
                    Endpoint Scopes (comma separated)
                  </label>
                  <input
                    type="text"
                    value={scopesStr}
                    onChange={(e) => setScopesStr(e.target.value)}
                    placeholder="* or GET /api/v1/users, POST /api/v1/*"
                    className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 text-sm font-mono focus:outline-none focus:border-blue-500"
                  />
                  <p className="text-[11px] text-slate-500 mt-1">
                    Use &apos;*&apos; for all routes, or specify methods and paths.
                  </p>
                </div>

                <div className="flex justify-end space-x-3 pt-3">
                  <button
                    type="button"
                    onClick={() => setShowCreateModal(false)}
                    className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm font-medium"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={creating}
                    className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium shadow-lg shadow-blue-600/20 disabled:opacity-50"
                  >
                    {creating ? "Generating..." : "Generate Secret Key"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Modal: Reveal Secret Key (Shown ONCE) */}
        {revealedKey && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/90 backdrop-blur-md">
            <div className="w-full max-w-lg bg-slate-900 border border-emerald-500/30 rounded-2xl p-6 shadow-2xl">
              <div className="flex items-center space-x-3 mb-4">
                <div className="p-2.5 rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                  <ShieldAlert className="w-6 h-6" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-white">Save Your API Key</h2>
                  <p className="text-xs text-slate-400">
                    This secret token will only be shown to you once.
                  </p>
                </div>
              </div>

              <div className="p-3.5 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-300 text-xs mb-4 flex items-start space-x-2">
                <AlertTriangle className="w-4 h-4 flex-shrink-0 mt-0.5 text-amber-400" />
                <span>
                  Please copy your API key and store it in a secure password manager. If you lose it,
                  you will need to generate a new key.
                </span>
              </div>

              <div className="relative mb-6">
                <input
                  type="text"
                  readOnly
                  value={revealedKey}
                  className="w-full px-3.5 py-3 pr-24 rounded-lg bg-slate-950 border border-slate-700 text-emerald-400 font-mono text-xs focus:outline-none"
                />
                <button
                  onClick={handleCopy}
                  className="absolute right-2 top-2 bottom-2 px-3 rounded-md bg-blue-600 hover:bg-blue-500 text-white text-xs font-medium flex items-center space-x-1.5 transition-all shadow-md"
                >
                  {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                  <span>{copied ? "Copied!" : "Copy"}</span>
                </button>
              </div>

              <div className="flex justify-end">
                <button
                  onClick={() => setRevealedKey(null)}
                  className="px-5 py-2.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-white text-sm font-medium transition-all"
                >
                  I have saved this key safely
                </button>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

