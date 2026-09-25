import React, { useEffect, useState } from 'react';
import {
  Settings,
  User,
  Shield,
  Bell,
  HardDrive,
  Link2,
  Code,
  Sliders,
  CheckCircle2,
  Key,
  Copy,
  Check,
  RefreshCw,
  AlertTriangle,
  ExternalLink,
  Trash2
} from 'lucide-react';
import { api } from '../../lib/api';
import { CapabilityBadge } from '../common/CapabilityBadge';
import { NavRoute } from '../layout/Sidebar';

interface Props {
  onNavigate: (route: NavRoute) => void;
}

export const SettingsView: React.FC<Props> = ({ onNavigate }) => {
  const [settings, setSettings] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('General');
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [newKeyModal, setNewKeyModal] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [generatedSecret, setGeneratedSecret] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState(false);

  const loadSettings = async () => {
    try {
      setLoading(true);
      const res = await api.getSettings();
      setSettings(res.data);
    } catch (err) {
      console.error('Failed to load settings:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadSettings();
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.saveSettings(settings);
      setSavedSuccess(true);
      setTimeout(() => setSavedSuccess(false), 2500);
    } catch (err: any) {
      alert(`Save failed: ${err.message}`);
    }
  };

  const handleGenerateKey = async () => {
    if (!newKeyName.trim()) return;
    try {
      const res = await api.createApiKey(newKeyName.trim());
      setGeneratedSecret(res.secret);
      setSettings((prev: any) => ({
        ...prev,
        apiKeys: [...prev.apiKeys, res.data]
      }));
      setNewKeyName('');
    } catch (err: any) {
      alert(`Key generation failed: ${err.message}`);
    }
  };

  if (loading || !settings) {
    return (
      <div className="flex items-center justify-center h-96">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header matching setting.png */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
            <span>Platform Configuration</span>
            <span>·</span>
            <CapabilityBadge state="CONFIGURED" />
          </div>
          <h1 className="text-2xl font-extrabold text-white tracking-tight mt-1">Settings</h1>
          <p className="text-xs text-slate-400">
            Manage your account, platform preferences, security, and infrastructure settings.
          </p>
        </div>

        {/* Platform Status matching setting.png */}
        <div className="flex items-center gap-3 px-4 py-2 rounded-xl bg-[#0D1527] border border-emerald-500/40 shadow-lg">
          <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-pulse" />
          <div className="flex flex-col text-left">
            <span className="text-xs font-bold text-white leading-tight">All Systems Operational</span>
            <span className="text-[10px] text-slate-400 font-mono">Mesh telemetry healthy</span>
          </div>
        </div>
      </div>

      {/* Tabs Row matching setting.png */}
      <div className="flex items-center gap-1.5 border-b border-slate-800 pb-2 overflow-x-auto">
        {[
          'General',
          'Account',
          'Security',
          'Notifications',
          'Infrastructure',
          'Integrations',
          'API & Developer',
          'Appearance',
          'Advanced'
        ].map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-3.5 py-1.5 rounded-xl text-xs font-medium transition-all ${
              activeTab === tab
                ? 'bg-blue-600 text-white shadow-md'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Main Settings Grid matching setting.png */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left Column: General & Account Settings (5 cols) */}
        <div className="lg:col-span-5 space-y-6">
          {/* General Settings */}
          <form onSubmit={handleSave} className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4 shadow-xl">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-sm font-bold text-white flex items-center gap-2">
                  <Settings className="w-4 h-4 text-cyan-400" />
                  General Settings
                </h2>
                <p className="text-[11px] text-slate-400">Basic configuration for your Decentralized.Host account.</p>
              </div>
              <button
                type="submit"
                className="px-3 py-1.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-md flex items-center gap-1"
              >
                {savedSuccess ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : null}
                <span>{savedSuccess ? 'Saved!' : 'Save Changes'}</span>
              </button>
            </div>

            <div className="space-y-3 font-mono text-xs">
              <div className="space-y-1">
                <label className="text-slate-400 font-sans text-xs">Organization Name</label>
                <input
                  type="text"
                  value={settings.organizationName}
                  onChange={(e) => setSettings({ ...settings, organizationName: e.target.value })}
                  className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white focus:outline-none"
                />
              </div>

              <div className="space-y-1">
                <label className="text-slate-400 font-sans text-xs">Time Zone</label>
                <select
                  value={settings.timezone}
                  onChange={(e) => setSettings({ ...settings, timezone: e.target.value })}
                  className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white focus:outline-none"
                >
                  <option>(GMT+05:30) Asia/Kolkata</option>
                  <option>(GMT+00:00) UTC</option>
                  <option>(GMT-04:00) America/New_York</option>
                  <option>(GMT+01:00) Europe/Berlin</option>
                </select>
              </div>

              <div className="space-y-1">
                <label className="text-slate-400 font-sans text-xs">Language</label>
                <select
                  value={settings.language}
                  onChange={(e) => setSettings({ ...settings, language: e.target.value })}
                  className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white focus:outline-none"
                >
                  <option>English (Default)</option>
                  <option>German</option>
                  <option>Spanish</option>
                </select>
              </div>

              <div className="space-y-1">
                <label className="text-slate-400 font-sans text-xs">Date Format</label>
                <input
                  type="text"
                  value={settings.dateFormat}
                  onChange={(e) => setSettings({ ...settings, dateFormat: e.target.value })}
                  className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white focus:outline-none"
                />
              </div>
            </div>
          </form>

          {/* Account Settings matching setting.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4 shadow-xl">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-sm font-bold text-white flex items-center gap-2">
                  <User className="w-4 h-4 text-purple-400" />
                  Account Settings
                </h2>
                <p className="text-[11px] text-slate-400">Manage personal info and credentials.</p>
              </div>
              <button
                onClick={() => alert('Password update challenge sent to verified keyring.')}
                className="px-2.5 py-1 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold"
              >
                Change Password
              </button>
            </div>

            <div className="flex items-center gap-3 p-3 rounded-xl bg-slate-900 border border-slate-800">
              <div className="w-10 h-10 rounded-full bg-gradient-to-tr from-blue-600 to-cyan-500 flex items-center justify-center font-bold text-white text-sm">
                F
              </div>
              <div>
                <div className="text-xs font-bold text-white flex items-center gap-2">
                  <span>{settings.account.fullName}</span>
                  <span className="px-1.5 py-0.5 rounded bg-amber-950 text-amber-400 border border-amber-500/30 text-[10px] font-mono">
                    Owner
                  </span>
                </div>
                <div className="text-[11px] text-slate-400 font-mono">{settings.account.email}</div>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 text-xs font-mono">
              <div className="space-y-1">
                <label className="text-slate-400 font-sans text-xs">Phone Number</label>
                <input
                  type="text"
                  value={settings.account.phone}
                  readOnly
                  className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-300 focus:outline-none"
                />
              </div>
              <div className="space-y-1">
                <label className="text-slate-400 font-sans text-xs">Job Title</label>
                <input
                  type="text"
                  value={settings.account.jobTitle}
                  readOnly
                  className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-slate-300 focus:outline-none"
                />
              </div>
            </div>
          </div>
        </div>

        {/* Center Column: Appearance, Notifications & Danger Zone (4 cols) */}
        <div className="lg:col-span-4 space-y-6">
          {/* Appearance & Theme matching setting.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-4">
            <h2 className="text-sm font-bold text-white flex items-center gap-2">
              <Sliders className="w-4 h-4 text-cyan-400" />
              Appearance & Theme
            </h2>

            <div className="grid grid-cols-3 gap-2">
              <div className="p-3 rounded-xl bg-slate-900/40 border border-slate-800 text-center opacity-60">
                <div className="text-xs font-semibold text-white">Light</div>
                <div className="text-[9px] text-slate-400">Clean & bright</div>
              </div>
              <div className="p-3 rounded-xl bg-blue-950/40 border border-blue-500 text-center shadow-lg ring-1 ring-blue-500">
                <div className="text-xs font-bold text-white">Dark ✓</div>
                <div className="text-[9px] text-cyan-300">Modern & easy</div>
              </div>
              <div className="p-3 rounded-xl bg-slate-900/40 border border-slate-800 text-center opacity-60">
                <div className="text-xs font-semibold text-white">System</div>
                <div className="text-[9px] text-slate-400">Follow OS</div>
              </div>
            </div>

            <div className="space-y-2 pt-2 border-t border-slate-800">
              <span className="text-xs font-semibold text-slate-300">Density</span>
              <select
                value={settings.density}
                onChange={(e) => setSettings({ ...settings, density: e.target.value })}
                className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none"
              >
                <option>Comfortable (Default)</option>
                <option>Compact</option>
              </select>
            </div>
          </div>

          {/* Notifications Toggles matching setting.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-3.5">
            <h2 className="text-sm font-bold text-white flex items-center gap-2">
              <Bell className="w-4 h-4 text-amber-400" />
              Notifications
            </h2>

            {[
              { key: 'deploymentUpdates', label: 'Deployment Updates', desc: 'Get notified when deployments complete or fail' },
              { key: 'nodeAlerts', label: 'Node Alerts', desc: 'Receive alerts about node status and performance' },
              { key: 'billingInvoices', label: 'Billing & Invoices', desc: 'Quota warnings and usage accounting alerts' },
              { key: 'securityAlerts', label: 'Security Alerts', desc: 'Important security events and access notifications' }
            ].map((item) => {
              const checked = (settings.notifications as any)[item.key];
              return (
                <div key={item.key} className="flex items-center justify-between">
                  <div className="flex flex-col">
                    <span className="text-xs font-semibold text-white">{item.label}</span>
                    <span className="text-[10px] text-slate-400">{item.desc}</span>
                  </div>
                  <button
                    onClick={() =>
                      setSettings({
                        ...settings,
                        notifications: { ...settings.notifications, [item.key]: !checked }
                      })
                    }
                    className={`w-9 h-5 rounded-full transition-colors relative ${
                      checked ? 'bg-blue-600' : 'bg-slate-800'
                    }`}
                  >
                    <span
                      className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
                        checked ? 'right-0.5' : 'left-0.5'
                      }`}
                    />
                  </button>
                </div>
              );
            })}
          </div>

          {/* Danger Zone matching setting.png */}
          <div className="rounded-2xl bg-rose-950/20 border border-rose-500/30 p-5 space-y-3">
            <div className="flex items-center gap-2 text-rose-400 font-bold text-sm">
              <AlertTriangle className="w-4 h-4" />
              <span>Danger Zone</span>
            </div>
            <p className="text-[11px] text-slate-400">These actions are irreversible and require operator confirmation.</p>

            <div className="flex items-center justify-between pt-1">
              <span className="text-xs text-slate-300">Export Account Data</span>
              <button
                onClick={() => alert('Exporting signed bundle with qualification evidence...')}
                className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold"
              >
                Export Data
              </button>
            </div>

            <div className="flex items-center justify-between pt-1">
              <span className="text-xs text-rose-400">Delete Account</span>
              <button
                onClick={() => alert('Account deletion blocked: Active workloads are currently bound to decentralized routing.')}
                className="px-3 py-1.5 rounded-lg bg-rose-950 text-rose-300 border border-rose-500/40 text-xs font-semibold"
              >
                Delete Account
              </button>
            </div>
          </div>
        </div>

        {/* Right Column: Infrastructure & Integrations & API Keys (3 cols) */}
        <div className="lg:col-span-3 space-y-6">
          {/* Infrastructure Preferences matching setting.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-3">
            <h2 className="text-sm font-bold text-white flex items-center gap-2">
              <HardDrive className="w-4 h-4 text-blue-400" />
              Infrastructure Preferences
            </h2>

            <div className="space-y-2 text-xs font-mono">
              <div className="space-y-1">
                <span className="text-slate-400 font-sans">Default Region</span>
                <select className="w-full px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-white">
                  <option>Auto (Best Performance)</option>
                  <option>North America</option>
                  <option>Europe</option>
                  <option>Asia</option>
                </select>
              </div>

              <div className="space-y-1">
                <span className="text-slate-400 font-sans">Default Storage Class</span>
                <select className="w-full px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-white">
                  <option>Balanced (Recommended)</option>
                  <option>High Performance</option>
                  <option>Cold Archive</option>
                </select>
              </div>
            </div>
          </div>

          {/* Integrations matching setting.png */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-3 font-mono text-xs">
            <h2 className="text-sm font-bold text-white font-sans flex items-center gap-2">
              <Link2 className="w-4 h-4 text-cyan-400" />
              Integrations
            </h2>

            <div className="space-y-2">
              <div className="flex items-center justify-between p-2 rounded-lg bg-slate-900 border border-slate-800">
                <span className="text-slate-300">GitHub</span>
                <span className="text-emerald-400 text-[10px] font-bold">Connected</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-lg bg-slate-900 border border-slate-800">
                <span className="text-slate-300">Cloudflare</span>
                <span className="text-slate-500 text-[10px]">Not Connected</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-lg bg-slate-900 border border-slate-800">
                <span className="text-slate-300">Discord</span>
                <span className="text-emerald-400 text-[10px] font-bold">Connected</span>
              </div>
              <div className="flex items-center justify-between p-2 rounded-lg bg-slate-900 border border-slate-800">
                <span className="text-slate-300">Slack</span>
                <span className="text-slate-500 text-[10px]">Not Connected</span>
              </div>
            </div>
          </div>

          {/* API Keys */}
          <div className="rounded-2xl bg-[#0D1527] border border-slate-800/80 p-5 space-y-3 font-mono text-xs">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-bold text-white font-sans flex items-center gap-2">
                <Key className="w-4 h-4 text-purple-400" />
                API Keys
              </h2>
              <button
                onClick={() => setNewKeyModal(true)}
                className="text-[11px] text-cyan-400 hover:underline"
              >
                + Generate
              </button>
            </div>

            <div className="space-y-2">
              {settings.apiKeys.map((key: any) => (
                <div key={key.id} className="p-2 rounded-lg bg-slate-900 border border-slate-800 space-y-0.5">
                  <div className="font-bold text-white truncate">{key.name}</div>
                  <div className="text-[10px] text-cyan-400 font-mono">{key.prefix}••••••••</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Generate API Key Modal */}
      {newKeyModal && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-md flex items-center justify-center p-4">
          <div className="w-full max-w-md rounded-2xl bg-[#0B1120] border border-slate-700 p-6 space-y-4">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <h3 className="text-base font-bold text-white">Generate Authority API Key</h3>
              <button
                onClick={() => {
                  setNewKeyModal(false);
                  setGeneratedSecret(null);
                }}
                className="text-slate-400 hover:text-white"
              >
                ✕
              </button>
            </div>

            {!generatedSecret ? (
              <div className="space-y-3">
                <div className="space-y-1">
                  <label className="text-xs text-slate-300">Key Name</label>
                  <input
                    type="text"
                    placeholder="e.g. Terraform CI Runner"
                    value={newKeyName}
                    onChange={(e) => setNewKeyName(e.target.value)}
                    className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-slate-800 text-white text-xs font-mono focus:outline-none"
                  />
                </div>
                <button
                  onClick={handleGenerateKey}
                  className="w-full py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold"
                >
                  Generate Key
                </button>
              </div>
            ) : (
              <div className="space-y-3 font-mono text-xs">
                <div className="p-3 rounded-xl bg-amber-950/40 border border-amber-500/40 text-amber-300 text-[11px]">
                  Copy this key now. It will never be shown again and cannot be recovered.
                </div>
                <div className="p-3 rounded-xl bg-black border border-slate-800 text-cyan-400 break-all select-all flex items-center justify-between">
                  <span>{generatedSecret}</span>
                  <button
                    onClick={() => {
                      navigator.clipboard.writeText(generatedSecret);
                      setCopiedKey(true);
                      setTimeout(() => setCopiedKey(false), 2000);
                    }}
                    className="p-1.5 rounded bg-slate-800 text-white hover:bg-slate-700 ml-2"
                  >
                    {copiedKey ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  </button>
                </div>
                <button
                  onClick={() => {
                    setNewKeyModal(false);
                    setGeneratedSecret(null);
                  }}
                  className="w-full py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold"
                >
                  Done
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
