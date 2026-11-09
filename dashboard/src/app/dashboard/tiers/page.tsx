"use client";

import { useEffect, useState } from "react";
import Navbar from "@/components/Navbar";
import { apiRequest } from "@/lib/api";
import { Sliders, Edit3, Check, X, Shield, Zap } from "lucide-react";

interface Tier {
  id: string;
  name: string;
  description: string;
  rate_limit_per_min: number;
  rate_limit_per_sec: number;
  monthly_quota: number;
}

export default function TiersPage() {
  const [tiers, setTiers] = useState<Tier[]>([]);
  const [loading, setLoading] = useState(true);

  // Edit Modal State
  const [editingTier, setEditingTier] = useState<Tier | null>(null);
  const [rpm, setRpm] = useState(60);
  const [rps, setRps] = useState(10);
  const [monthlyQuota, setMonthlyQuota] = useState(10000);
  const [description, setDescription] = useState("");
  const [saving, setSaving] = useState(false);

  const fetchTiers = async () => {
    try {
      setLoading(true);
      const data = await apiRequest<Tier[]>("/api/tiers");
      setTiers(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTiers();
  }, []);

  const openEdit = (tier: Tier) => {
    setEditingTier(tier);
    setRpm(tier.rate_limit_per_min);
    setRps(tier.rate_limit_per_sec);
    setMonthlyQuota(tier.monthly_quota);
    setDescription(tier.description || "");
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingTier) return;
    setSaving(true);

    try {
      await apiRequest(`/api/tiers/${editingTier.id}`, {
        method: "PUT",
        body: JSON.stringify({
          rate_limit_per_min: Number(rpm),
          rate_limit_per_sec: Number(rps),
          monthly_quota: Number(monthlyQuota),
          description,
        }),
      });

      setEditingTier(null);
      await fetchTiers();
    } catch (err: any) {
      alert("Failed to update tier: " + err.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <Navbar />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center space-x-2">
            <Sliders className="w-6 h-6 text-blue-400" />
            <span>Tiers & Rate Limits</span>
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Configure minute limits, burst rates, and monthly billing quotas per tier
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {tiers.map((tier) => (
            <div
              key={tier.id}
              className="bg-slate-900/60 border border-slate-800 rounded-xl p-6 relative hover:border-slate-700 transition-all flex flex-col justify-between shadow-lg"
            >
              <div>
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center space-x-2">
                    <Zap className="w-5 h-5 text-blue-400" />
                    <h2 className="text-lg font-bold text-white">{tier.name}</h2>
                  </div>
                  <button
                    onClick={() => openEdit(tier)}
                    className="p-1.5 rounded-lg text-slate-400 hover:text-blue-400 hover:bg-slate-800 transition-colors"
                    title="Edit tier limits"
                  >
                    <Edit3 className="w-4 h-4" />
                  </button>
                </div>

                <p className="text-xs text-slate-400 mb-6 min-h-[36px]">
                  {tier.description || "General tier configuration"}
                </p>

                <div className="space-y-3 font-mono text-xs">
                  <div className="flex justify-between p-2.5 rounded-lg bg-slate-950 border border-slate-800/80">
                    <span className="text-slate-400 font-sans">Rate Limit (RPM):</span>
                    <span className="text-blue-400 font-bold">{tier.rate_limit_per_min} req/min</span>
                  </div>
                  <div className="flex justify-between p-2.5 rounded-lg bg-slate-950 border border-slate-800/80">
                    <span className="text-slate-400 font-sans">Burst Limit (RPS):</span>
                    <span className="text-cyan-400 font-bold">{tier.rate_limit_per_sec} req/sec</span>
                  </div>
                  <div className="flex justify-between p-2.5 rounded-lg bg-slate-950 border border-slate-800/80">
                    <span className="text-slate-400 font-sans">Monthly Quota:</span>
                    <span className="text-emerald-400 font-bold">{tier.monthly_quota.toLocaleString()} calls</span>
                  </div>
                </div>
              </div>

              <div className="mt-6 pt-4 border-t border-slate-800/80 text-[11px] text-slate-500">
                Tier ID: <span className="font-mono text-slate-400">{tier.id}</span>
              </div>
            </div>
          ))}
        </div>

        {/* Edit Tier Modal */}
        {editingTier && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm">
            <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-2xl">
              <div className="flex items-center justify-between mb-5">
                <h2 className="text-lg font-bold text-white">Edit Tier: {editingTier.name}</h2>
                <button onClick={() => setEditingTier(null)} className="text-slate-400 hover:text-white">
                  <X className="w-5 h-5" />
                </button>
              </div>

              <form onSubmit={handleSave} className="space-y-4">
                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1.5">
                    Description
                  </label>
                  <input
                    type="text"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1.5">
                    Rate Limit (Requests per minute)
                  </label>
                  <input
                    type="number"
                    required
                    min={1}
                    value={rpm}
                    onChange={(e) => setRpm(Number(e.target.value))}
                    className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1.5">
                    Burst Limit (Requests per second)
                  </label>
                  <input
                    type="number"
                    required
                    min={1}
                    value={rps}
                    onChange={(e) => setRps(Number(e.target.value))}
                    className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-slate-300 mb-1.5">
                    Monthly Quota (Total calls per calendar month)
                  </label>
                  <input
                    type="number"
                    required
                    min={100}
                    value={monthlyQuota}
                    onChange={(e) => setMonthlyQuota(Number(e.target.value))}
                    className="w-full px-3.5 py-2.5 rounded-lg bg-slate-950 border border-slate-800 text-slate-100 text-sm focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div className="flex justify-end space-x-3 pt-3">
                  <button
                    type="button"
                    onClick={() => setEditingTier(null)}
                    className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-sm font-medium"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={saving}
                    className="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium shadow-lg shadow-blue-600/20 disabled:opacity-50"
                  >
                    {saving ? "Saving..." : "Save Changes"}
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

