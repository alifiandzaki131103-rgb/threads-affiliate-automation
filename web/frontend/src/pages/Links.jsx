import { useState, useEffect } from 'react';
import { Plus, Trash2, ExternalLink, Copy } from 'lucide-react';
import api from '../api';

export default function Links() {
  const [links, setLinks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [bulkUrls, setBulkUrls] = useState('');
  const [adding, setAdding] = useState(false);
  const [message, setMessage] = useState('');

  async function loadLinks() {
    try {
      const { data } = await api.get('/links');
      setLinks(data.links || []);
    } catch (err) {
      console.error('Failed to load links:', err);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void Promise.resolve().then(loadLinks);
  }, []);

  const handleBulkAdd = async (e) => {
    e.preventDefault();
    setAdding(true);
    setMessage('');

    const urls = bulkUrls.split('\n').map(u => u.trim()).filter(u => u);
    if (urls.length === 0) {
      setMessage('Masukkan minimal 1 URL');
      setAdding(false);
      return;
    }

    try {
      const { data } = await api.post('/links/bulk', { urls });
      setMessage(`✅ ${data.count} links berhasil ditambahkan!`);
      setBulkUrls('');
      setShowAdd(false);
      loadLinks();
    } catch (err) {
      setMessage(`❌ ${err.response?.data?.error || 'Failed to add links'}`);
    } finally {
      setAdding(false);
    }
  };

  const copySlug = (slug) => {
    navigator.clipboard.writeText(`${window.location.origin}/s/${slug}`);
  };

  const handleDelete = async (id) => {
    if (!confirm('Hapus link ini?')) return;
    try {
      await api.delete(`/links/${id}`);
      loadLinks();
    } catch (err) {
      alert('Gagal hapus link');
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-white">Affiliate Links</h2>
        <button
          onClick={() => setShowAdd(!showAdd)}
          className="flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition"
        >
          <Plus size={16} />
          Add Links
        </button>
      </div>

      {message && (
        <div className="bg-gray-800 border border-gray-700 text-gray-200 px-4 py-2 rounded-lg mb-4 text-sm">
          {message}
        </div>
      )}

      {/* Add Links Form */}
      {showAdd && (
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-6 mb-6">
          <h3 className="text-lg font-semibold text-white mb-3">Bulk Add Links</h3>
          <p className="text-sm text-gray-400 mb-4">
            Paste affiliate links dari Shopee/TikTok (satu per baris). Auto-detect platform.
          </p>
          <form onSubmit={handleBulkAdd}>
            <textarea
              value={bulkUrls}
              onChange={(e) => setBulkUrls(e.target.value)}
              placeholder={"https://s.shopee.co.id/abc123\nhttps://vt.tiktok.com/xyz789\nhttps://s.shopee.co.id/def456"}
              className="w-full h-32 bg-gray-800 border border-gray-700 rounded-lg px-4 py-3 text-white text-sm font-mono focus:outline-none focus:border-indigo-500 resize-none"
            />
            <div className="flex gap-3 mt-3">
              <button
                type="submit"
                disabled={adding}
                className="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50"
              >
                {adding ? 'Adding...' : 'Add All Links'}
              </button>
              <button
                type="button"
                onClick={() => setShowAdd(false)}
                className="bg-gray-700 hover:bg-gray-600 text-white px-4 py-2 rounded-lg text-sm transition"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Links Table */}
      <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-400">Loading...</div>
        ) : links.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            <p>Belum ada links. Klik "Add Links" untuk mulai.</p>
          </div>
        ) : (
          <table className="w-full">
            <thead className="bg-gray-800/50">
              <tr>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-400 uppercase">Platform</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-400 uppercase">Short URL</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-400 uppercase">Original</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-400 uppercase">Clicks</th>
                <th className="text-left px-4 py-3 text-xs font-medium text-gray-400 uppercase">Status</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800">
              {links.map((link) => (
                <tr key={link.id} className="hover:bg-gray-800/30">
                  <td className="px-4 py-3">
                    <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${
                      link.platform === 'shopee' ? 'bg-orange-900/30 text-orange-300' :
                      link.platform === 'tiktok' ? 'bg-pink-900/30 text-pink-300' :
                      'bg-gray-700 text-gray-300'
                    }`}>
                      {link.platform}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => copySlug(link.short_slug)}
                      className="flex items-center gap-1 text-sm text-indigo-400 hover:text-indigo-300"
                    >
                      <Copy size={12} />
                      /s/{link.short_slug}
                    </button>
                  </td>
                  <td className="px-4 py-3">
                    <a
                      href={link.original_url}
                      target="_blank"
                      rel="noopener"
                      className="flex items-center gap-1 text-sm text-gray-300 hover:text-white truncate max-w-[200px]"
                    >
                      {link.original_url}
                      <ExternalLink size={12} />
                    </a>
                  </td>
                  <td className="px-4 py-3 text-sm text-white font-medium">
                    {link.click_count || 0}
                  </td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex px-2 py-0.5 rounded text-xs ${
                      link.status === 'active' ? 'bg-green-900/30 text-green-300' : 'bg-red-900/30 text-red-300'
                    }`}>
                      {link.status}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <button onClick={() => handleDelete(link.id)} className="text-gray-500 hover:text-red-400 transition">
                      <Trash2 size={16} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
