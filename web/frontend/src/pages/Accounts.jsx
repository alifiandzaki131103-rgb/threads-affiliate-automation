import { useState, useEffect } from 'react';
import { User, Shield, Zap, AlertTriangle, Settings, Trash2, Plus, Eye, EyeOff } from 'lucide-react';
import api from '../api';

const PERSONAS = ['honest_friend', 'hot_take', 'problem_solver', 'curious_explorer', 'lifestyle_sharer', 'comparison_nerd'];

const STATUS_CONFIG = {
  active: { color: 'bg-green-400', label: 'Active', textColor: 'text-green-400', bgBadge: 'bg-green-400/10 border-green-400/20' },
  paused: { color: 'bg-yellow-400', label: 'Paused', textColor: 'text-yellow-400', bgBadge: 'bg-yellow-400/10 border-yellow-400/20' },
  flagged: { color: 'bg-red-400', label: 'Flagged', textColor: 'text-red-400', bgBadge: 'bg-red-400/10 border-red-400/20' },
};

function StatusBadge({ status }) {
  const config = STATUS_CONFIG[status] || STATUS_CONFIG.active;
  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${config.bgBadge} ${config.textColor}`}>
      <span className={`w-2 h-2 rounded-full ${config.color}`}></span>
      {config.label}
    </span>
  );
}

function PersonaBadge({ persona }) {
  return (
    <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium bg-indigo-400/10 border border-indigo-400/20 text-indigo-400">
      <User size={12} />
      {persona?.replace(/_/g, ' ') || 'No persona'}
    </span>
  );
}

function Modal({ open, onClose, title, children }) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose}></div>
      <div className="relative bg-gray-900 border border-gray-800 rounded-xl w-full max-w-lg p-6 shadow-2xl">
        <h3 className="text-lg font-semibold text-white mb-4">{title}</h3>
        {children}
      </div>
    </div>
  );
}

export default function Accounts() {
  const [accounts, setAccounts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showConnectModal, setShowConnectModal] = useState(false);
  const [editAccount, setEditAccount] = useState(null);
  const [saving, setSaving] = useState(false);

  // Connect form state
  const [connectForm, setConnectForm] = useState({
    threads_user_id: '',
    access_token: '',
    persona: 'honest_friend',
    niche: '',
  });
  const [showToken, setShowToken] = useState(false);

  // Edit form state
  const [editForm, setEditForm] = useState({
    persona: '',
    niche: '',
    auto_mode: false,
    max_daily_posts: 15,
  });

  async function loadAccounts() {
    try {
      const { data } = await api.get('/accounts');
      setAccounts(data.accounts || data || []);
    } catch (err) {
      console.error('Failed to load accounts:', err);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { loadAccounts(); }, []);

  const handleConnect = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      await api.post('/accounts', connectForm);
      setShowConnectModal(false);
      setConnectForm({ threads_user_id: '', access_token: '', persona: 'honest_friend', niche: '' });
      setShowToken(false);
      loadAccounts();
    } catch (err) {
      alert('Failed to add account: ' + (err.response?.data?.error || err.message));
    } finally {
      setSaving(false);
    }
  };

  const handleEdit = async (e) => {
    e.preventDefault();
    if (!editAccount) return;
    setSaving(true);
    try {
      await api.put(`/accounts/${editAccount.id}`, {
        persona: editForm.persona,
        niche: editForm.niche,
        auto_mode: editForm.auto_mode,
        max_daily_posts: editForm.max_daily_posts,
      });
      setEditAccount(null);
      loadAccounts();
    } catch (err) {
      alert('Failed to update account: ' + (err.response?.data?.error || err.message));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this account? This action cannot be undone.')) return;
    try {
      await api.delete(`/accounts/${id}`);
      loadAccounts();
    } catch {
      alert('Failed to delete account');
    }
  };

  const handleToggleAutoMode = async (account) => {
    try {
      await api.put(`/accounts/${account.id}`, {
        auto_mode: !account.auto_mode,
      });
      loadAccounts();
    } catch {
      alert('Failed to toggle auto mode');
    }
  };

  const handleToggleStatus = async (account) => {
    const newStatus = account.status === 'active' ? 'paused' : 'active';
    try {
      await api.put(`/accounts/${account.id}`, { status: newStatus });
      loadAccounts();
    } catch {
      alert('Failed to update status');
    }
  };

  const openEditModal = (account) => {
    setEditAccount(account);
    setEditForm({
      persona: account.persona || 'honest_friend',
      niche: account.niche || '',
      auto_mode: account.auto_mode ?? true,
      max_daily_posts: account.max_daily_posts || 15,
    });
  };

  const tokenHint = connectForm.access_token.length > 0 && (
    !connectForm.access_token.startsWith('THQ') && !connectForm.access_token.startsWith('EA')
  );

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-white">Threads Accounts</h2>
          <p className="text-gray-500 text-sm mt-1">{accounts.length} account{accounts.length !== 1 ? 's' : ''} connected</p>
        </div>
        <button
          onClick={() => setShowConnectModal(true)}
          className="flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2.5 rounded-xl transition text-sm font-medium shadow-lg shadow-indigo-600/20"
        >
          <Plus size={16} /> Connect Account
        </button>
      </div>

      {/* Flagged accounts warning banner */}
      {accounts.filter(a => a.status === 'flagged').length > 0 && (
        <div className="mb-6 bg-red-500/10 border border-red-500/20 rounded-xl p-4 flex items-start gap-3">
          <AlertTriangle className="text-red-400 shrink-0 mt-0.5" size={20} />
          <div>
            <p className="text-red-400 font-medium text-sm">Circuit Breaker Triggered</p>
            <p className="text-red-400/70 text-xs mt-1">
              {accounts.filter(a => a.status === 'flagged').length} account(s) flagged due to rate limiting or errors. 
              These accounts are in cooldown and won't post until the circuit resets.
            </p>
          </div>
        </div>
      )}

      {/* Account Cards */}
      {loading ? (
        <div className="text-center text-gray-400 py-12">
          <div className="animate-spin w-8 h-8 border-2 border-gray-600 border-t-indigo-500 rounded-full mx-auto mb-3"></div>
          Loading accounts...
        </div>
      ) : accounts.length === 0 ? (
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-12 text-center">
          <User className="mx-auto text-gray-600 mb-3" size={40} />
          <p className="text-gray-400 font-medium">No Threads accounts connected</p>
          <p className="text-gray-500 text-sm mt-1">Connect an account to start auto-posting affiliate content.</p>
          <button
            onClick={() => setShowConnectModal(true)}
            className="mt-4 inline-flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg transition text-sm"
          >
            <Plus size={14} /> Connect Your First Account
          </button>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {accounts.map(account => (
            <div key={account.id} className="bg-gray-900 rounded-xl border border-gray-800 p-5 flex flex-col">
              {/* Flagged banner inside card */}
              {account.status === 'flagged' && (
                <div className="mb-3 bg-red-500/10 border border-red-500/20 rounded-lg p-3 flex items-center gap-2">
                  <AlertTriangle className="text-red-400 shrink-0" size={14} />
                  <div className="text-xs">
                    <span className="text-red-400 font-medium">Circuit breaker active</span>
                    {account.flagged_count && (
                      <span className="text-red-400/60 ml-2">• Flagged {account.flagged_count}x</span>
                    )}
                    {account.cooldown_until && (
                      <span className="text-red-400/60 ml-2">• Cooldown until {new Date(account.cooldown_until).toLocaleTimeString()}</span>
                    )}
                  </div>
                </div>
              )}

              {/* Top row: ID + Status */}
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-gray-800 flex items-center justify-center">
                    <User className="text-gray-400" size={20} />
                  </div>
                  <div>
                    <p className="text-white font-medium text-sm">
                      {account.username || `ID: ${account.threads_user_id}`}
                    </p>
                    {account.username && (
                      <p className="text-gray-500 text-xs">ID: {account.threads_user_id}</p>
                    )}
                  </div>
                </div>
                <StatusBadge status={account.status || 'active'} />
              </div>

              {/* Badges row */}
              <div className="flex flex-wrap gap-2 mb-4">
                <PersonaBadge persona={account.persona} />
                {account.niche && (
                  <span className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-gray-800 border border-gray-700 text-gray-300">
                    {account.niche}
                  </span>
                )}
              </div>

              {/* Stats row */}
              <div className="flex items-center gap-4 mb-4 text-xs text-gray-400">
                <div className="flex items-center gap-1.5">
                  <Zap size={12} className="text-yellow-400" />
                  <span>{account.daily_post_count || 0} / {account.max_daily_posts || 15} posts today</span>
                </div>
                {account.flagged_count > 0 && (
                  <div className="flex items-center gap-1.5">
                    <Shield size={12} className="text-red-400" />
                    <span>{account.flagged_count} flags</span>
                  </div>
                )}
              </div>

              {/* Auto-mode toggle + Actions */}
              <div className="flex items-center justify-between mt-auto pt-3 border-t border-gray-800">
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleToggleAutoMode(account)}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                      account.auto_mode ? 'bg-indigo-600' : 'bg-gray-700'
                    }`}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        account.auto_mode ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                  <span className="text-xs text-gray-400">Auto-mode</span>
                </div>

                <div className="flex items-center gap-1">
                  <button
                    onClick={() => openEditModal(account)}
                    className="p-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition"
                    title="Edit account"
                  >
                    <Settings size={15} />
                  </button>
                  <button
                    onClick={() => handleToggleStatus(account)}
                    className={`p-2 rounded-lg transition ${
                      account.status === 'active'
                        ? 'text-yellow-400 hover:text-yellow-300 hover:bg-yellow-400/10'
                        : 'text-green-400 hover:text-green-300 hover:bg-green-400/10'
                    }`}
                    title={account.status === 'active' ? 'Pause account' : 'Resume account'}
                  >
                    <Shield size={15} />
                  </button>
                  <button
                    onClick={() => handleDelete(account.id)}
                    className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-400/10 rounded-lg transition"
                    title="Delete account"
                  >
                    <Trash2 size={15} />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Connect Account Modal */}
      <Modal open={showConnectModal} onClose={() => setShowConnectModal(false)} title="Connect Threads Account">
        <form onSubmit={handleConnect} className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1.5">Threads User ID</label>
            <input
              type="text"
              value={connectForm.threads_user_id}
              onChange={e => setConnectForm({ ...connectForm, threads_user_id: e.target.value })}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-white text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition"
              placeholder="Your numeric Threads user ID"
              required
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1.5">Access Token</label>
            <div className="relative">
              <input
                type={showToken ? 'text' : 'password'}
                value={connectForm.access_token}
                onChange={e => setConnectForm({ ...connectForm, access_token: e.target.value })}
                className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 pr-10 text-white text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition"
                placeholder="From Meta Developer Portal"
                required
              />
              <button
                type="button"
                onClick={() => setShowToken(!showToken)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition"
              >
                {showToken ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
            {tokenHint && (
              <p className="text-yellow-400/80 text-xs mt-1.5 flex items-center gap-1">
                <AlertTriangle size={11} />
                Token should start with "THQ" or "EA" and be ~200 characters
              </p>
            )}
            {!tokenHint && connectForm.access_token.length === 0 && (
              <p className="text-gray-500 text-xs mt-1.5">
                Must start with THQ or EA, approximately 200 characters
              </p>
            )}
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1.5">Persona</label>
            <select
              value={connectForm.persona}
              onChange={e => setConnectForm({ ...connectForm, persona: e.target.value })}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-white text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition"
            >
              {PERSONAS.map(p => (
                <option key={p} value={p}>{p.replace(/_/g, ' ')}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1.5">Niche</label>
            <input
              type="text"
              value={connectForm.niche}
              onChange={e => setConnectForm({ ...connectForm, niche: e.target.value })}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-white text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition"
              placeholder="e.g. skincare, tech gadgets, fashion"
            />
          </div>

          <div className="flex items-center gap-3 pt-2">
            <button
              type="submit"
              disabled={saving}
              className="flex-1 bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2.5 rounded-lg text-sm font-medium disabled:opacity-50 transition"
            >
              {saving ? 'Connecting...' : 'Connect Account'}
            </button>
            <button
              type="button"
              onClick={() => setShowConnectModal(false)}
              className="px-4 py-2.5 rounded-lg text-sm text-gray-400 hover:text-white hover:bg-gray-800 transition"
            >
              Cancel
            </button>
          </div>
        </form>
      </Modal>

      {/* Edit Account Modal */}
      <Modal open={!!editAccount} onClose={() => setEditAccount(null)} title="Edit Account Settings">
        <form onSubmit={handleEdit} className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1.5">Persona</label>
            <select
              value={editForm.persona}
              onChange={e => setEditForm({ ...editForm, persona: e.target.value })}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-white text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition"
            >
              {PERSONAS.map(p => (
                <option key={p} value={p}>{p.replace(/_/g, ' ')}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1.5">Niche</label>
            <input
              type="text"
              value={editForm.niche}
              onChange={e => setEditForm({ ...editForm, niche: e.target.value })}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2.5 text-white text-sm focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition"
              placeholder="e.g. skincare, tech gadgets, fashion"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1.5">Auto Mode</label>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => setEditForm({ ...editForm, auto_mode: !editForm.auto_mode })}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  editForm.auto_mode ? 'bg-indigo-600' : 'bg-gray-700'
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    editForm.auto_mode ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
              <span className="text-sm text-gray-300">
                {editForm.auto_mode ? 'Enabled — posts automatically' : 'Disabled — manual only'}
              </span>
            </div>
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1.5">
              Max Daily Posts: <span className="text-white font-medium">{editForm.max_daily_posts}</span>
            </label>
            <input
              type="range"
              min="5"
              max="25"
              value={editForm.max_daily_posts}
              onChange={e => setEditForm({ ...editForm, max_daily_posts: parseInt(e.target.value) })}
              className="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-indigo-600"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>5</span>
              <span>15</span>
              <span>25</span>
            </div>
          </div>

          <div className="flex items-center gap-3 pt-2">
            <button
              type="submit"
              disabled={saving}
              className="flex-1 bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2.5 rounded-lg text-sm font-medium disabled:opacity-50 transition"
            >
              {saving ? 'Saving...' : 'Save Changes'}
            </button>
            <button
              type="button"
              onClick={() => setEditAccount(null)}
              className="px-4 py-2.5 rounded-lg text-sm text-gray-400 hover:text-white hover:bg-gray-800 transition"
            >
              Cancel
            </button>
          </div>
        </form>
      </Modal>

      {/* Help section */}
      <div className="mt-6 bg-gray-900/50 rounded-xl border border-gray-800 p-4">
        <h3 className="text-sm font-medium text-gray-300 mb-2">📋 How to Get Your Access Token</h3>
        <ol className="text-xs text-gray-500 space-y-1 list-decimal list-inside">
          <li>Go to <a href="https://developers.facebook.com" target="_blank" rel="noopener noreferrer" className="text-indigo-400 hover:underline">Meta Developer Portal</a></li>
          <li>Create an App → select "Business" type</li>
          <li>Add the "Threads API" product</li>
          <li>Generate a User Token with scopes: threads_basic, threads_content_publish</li>
          <li>Copy your User ID and Access Token into the connect form above</li>
        </ol>
      </div>
    </div>
  );
}
