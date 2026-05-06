import { useState, useEffect } from 'react';
import { UserCircle, Plus, Trash2, CheckCircle } from 'lucide-react';
import api from '../api';

export default function Accounts() {
  const [accounts, setAccounts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ threads_user_id: '', access_token: '', persona: 'honest_friend', niche: '' });
  const [saving, setSaving] = useState(false);

  async function loadAccounts() {
    try {
      const { data } = await api.get('/accounts');
      setAccounts(data.accounts || []);
    } catch (err) {
      console.error('Failed to load accounts:', err);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { void Promise.resolve().then(loadAccounts); }, []);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      await api.post('/accounts', form);
      setShowForm(false);
      setForm({ threads_user_id: '', access_token: '', persona: 'honest_friend', niche: '' });
      loadAccounts();
    } catch (err) {
      alert('Failed to add account: ' + (err.response?.data?.error || err.message));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this account?')) return;
    try {
      await api.delete(`/accounts/${id}`);
      loadAccounts();
    } catch {
      alert('Failed to delete account');
    }
  };

  const personas = ['honest_friend', 'hot_take', 'problem_solver', 'curious_explorer', 'lifestyle_sharer', 'comparison_nerd'];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-white">Threads Accounts</h2>
        <button onClick={() => setShowForm(!showForm)} className="flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg transition text-sm">
          <Plus size={16} /> Add Account
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="bg-gray-900 rounded-xl border border-gray-800 p-5 mb-6 space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Threads User ID</label>
            <input type="text" value={form.threads_user_id} onChange={e => setForm({...form, threads_user_id: e.target.value})}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm" placeholder="Your Threads numeric user ID" required />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Access Token</label>
            <input type="password" value={form.access_token} onChange={e => setForm({...form, access_token: e.target.value})}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm" placeholder="From Meta Developer Portal" required />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Persona</label>
            <select value={form.persona} onChange={e => setForm({...form, persona: e.target.value})}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm">
              {personas.map(p => <option key={p} value={p}>{p.replace('_', ' ')}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Niche</label>
            <input type="text" value={form.niche} onChange={e => setForm({...form, niche: e.target.value})}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm" placeholder="e.g. skincare, tech, fashion" />
          </div>
          <button type="submit" disabled={saving} className="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg text-sm disabled:opacity-50">
            {saving ? 'Saving...' : 'Connect Account'}
          </button>
        </form>
      )}

      {loading ? (
        <div className="text-center text-gray-400 py-8">Loading...</div>
      ) : accounts.length === 0 ? (
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center">
          <UserCircle className="mx-auto text-gray-600 mb-3" size={32} />
          <p className="text-gray-400">Belum ada akun Threads terhubung.</p>
          <p className="text-gray-500 text-sm mt-1">Tambahkan akun untuk mulai auto-posting.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {accounts.map(account => (
            <div key={account.id} className="bg-gray-900 rounded-xl border border-gray-800 p-4 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <CheckCircle className="text-green-400" size={20} />
                <div>
                  <p className="text-white text-sm font-medium">ID: {account.threads_user_id}</p>
                  <p className="text-gray-500 text-xs">{account.persona?.replace('_', ' ')} • {account.niche || 'No niche'}</p>
                </div>
              </div>
              <button onClick={() => handleDelete(account.id)} className="text-red-400 hover:text-red-300 p-2">
                <Trash2 size={16} />
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="mt-6 bg-gray-900/50 rounded-xl border border-gray-800 p-4">
        <h3 className="text-sm font-medium text-gray-300 mb-2">📋 Cara Mendapatkan Access Token</h3>
        <ol className="text-xs text-gray-500 space-y-1 list-decimal list-inside">
          <li>Buka <a href="https://developers.facebook.com" target="_blank" className="text-indigo-400 hover:underline">Meta Developer Portal</a></li>
          <li>Buat App → pilih "Business" type</li>
          <li>Tambahkan product "Threads API"</li>
          <li>Generate User Token dengan scope: threads_basic, threads_content_publish</li>
          <li>Copy User ID dan Access Token ke form di atas</li>
        </ol>
      </div>
    </div>
  );
}
