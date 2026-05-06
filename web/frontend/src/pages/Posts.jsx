import { useEffect, useState } from 'react';
import { CheckCircle, Clock, RefreshCw, Send, Sparkles, XCircle } from 'lucide-react';
import api from '../api';

const statusConfig = {
  published: { icon: <CheckCircle size={14} />, color: 'text-green-400', bg: 'bg-green-900/30' },
  approved: { icon: <Clock size={14} />, color: 'text-blue-400', bg: 'bg-blue-900/30' },
  pending_review: { icon: <Clock size={14} />, color: 'text-yellow-400', bg: 'bg-yellow-900/30' },
  failed: { icon: <XCircle size={14} />, color: 'text-red-400', bg: 'bg-red-900/30' },
  draft: { icon: <Clock size={14} />, color: 'text-gray-400', bg: 'bg-gray-800' },
};

const personaColors = {
  honest_friend: 'bg-blue-900/30 text-blue-300',
  hot_take: 'bg-red-900/30 text-red-300',
  problem_solver: 'bg-green-900/30 text-green-300',
  curious_explorer: 'bg-purple-900/30 text-purple-300',
  lifestyle_sharer: 'bg-pink-900/30 text-pink-300',
  comparison_nerd: 'bg-orange-900/30 text-orange-300',
};

export default function Posts() {
  const [posts, setPosts] = useState([]);
  const [links, setLinks] = useState([]);
  const [accounts, setAccounts] = useState([]);
  const [selectedLinkID, setSelectedLinkID] = useState('');
  const [selectedAccountID, setSelectedAccountID] = useState('');
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [busyPostID, setBusyPostID] = useState('');
  const [message, setMessage] = useState('');

  async function loadData() {
    setLoading(true);
    try {
      const [postsResponse, linksResponse, accountsResponse] = await Promise.all([
        api.get('/posts'),
        api.get('/links'),
        api.get('/accounts'),
      ]);

      const nextPosts = postsResponse.data.posts || [];
      const nextLinks = linksResponse.data.links || [];
      const nextAccounts = accountsResponse.data.accounts || [];

      setPosts(nextPosts);
      setLinks(nextLinks);
      setAccounts(nextAccounts);
      setSelectedLinkID((current) => current || nextLinks[0]?.id || '');
      setSelectedAccountID((current) => current || nextAccounts[0]?.id || '');
    } catch (err) {
      setMessage(`Failed to load posts: ${err.response?.data?.error || err.message}`);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void Promise.resolve().then(loadData);
  }, []);

  const generateContent = async (event) => {
    event.preventDefault();
    if (!selectedLinkID || !selectedAccountID) {
      setMessage('Choose one affiliate link and one Threads account first.');
      return;
    }

    setGenerating(true);
    setMessage('');
    try {
      await api.post('/posts/generate', { link_id: selectedLinkID, account_id: selectedAccountID });
      setMessage('Content generation queued. Refresh in a moment to see the new post.');
      await loadData();
    } catch (err) {
      setMessage(`Failed to queue generation: ${err.response?.data?.error || err.message}`);
    } finally {
      setGenerating(false);
    }
  };

  const approvePost = async (postID) => {
    setBusyPostID(postID);
    setMessage('');
    try {
      await api.post(`/posts/${postID}/approve`);
      setPosts((current) => current.map((post) => post.id === postID ? { ...post, status: 'approved' } : post));
      setMessage('Post approved.');
    } catch (err) {
      setMessage(`Failed to approve post: ${err.response?.data?.error || err.message}`);
    } finally {
      setBusyPostID('');
    }
  };

  const publishPost = async (postID) => {
    setBusyPostID(postID);
    setMessage('');
    try {
      await api.post(`/posts/${postID}/publish`);
      setMessage('Post queued for publishing.');
      await loadData();
    } catch (err) {
      setMessage(`Failed to publish post: ${err.response?.data?.error || err.message}`);
    } finally {
      setBusyPostID('');
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-white">Posts</h2>
        <div className="flex gap-2">
          <span className="text-sm text-gray-400 bg-gray-800 px-3 py-1 rounded-lg">
            {posts.filter((post) => post.status === 'published').length} published
          </span>
          <span className="text-sm text-gray-400 bg-gray-800 px-3 py-1 rounded-lg">
            {posts.filter((post) => post.status === 'approved').length} scheduled
          </span>
        </div>
      </div>

      {message && (
        <div className="bg-gray-800 border border-gray-700 text-gray-200 px-4 py-2 rounded-lg mb-4 text-sm">
          {message}
        </div>
      )}

      <form onSubmit={generateContent} className="bg-gray-900 rounded-xl border border-gray-800 p-5 mb-6">
        <div className="flex items-center gap-2 mb-4">
          <Sparkles className="text-indigo-400" size={18} />
          <h3 className="text-lg font-semibold text-white">Generate Content</h3>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <label className="block">
            <span className="block text-sm text-gray-400 mb-1">Affiliate Link</span>
            <select
              value={selectedLinkID}
              onChange={(event) => setSelectedLinkID(event.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-indigo-500"
            >
              <option value="">Select a link</option>
              {links.map((link) => (
                <option key={link.id} value={link.id}>
                  {link.platform} /go/{link.short_slug}
                </option>
              ))}
            </select>
          </label>
          <label className="block">
            <span className="block text-sm text-gray-400 mb-1">Threads Account</span>
            <select
              value={selectedAccountID}
              onChange={(event) => setSelectedAccountID(event.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-indigo-500"
            >
              <option value="">Select an account</option>
              {accounts.map((account) => (
                <option key={account.id} value={account.id}>
                  {account.threads_user_id} · {account.persona || 'default'}
                </option>
              ))}
            </select>
          </label>
        </div>
        <div className="flex gap-3 mt-4">
          <button
            type="submit"
            disabled={generating || !links.length || !accounts.length}
            className="inline-flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50"
          >
            <Sparkles size={16} />
            {generating ? 'Queueing...' : 'Generate Content'}
          </button>
          <button
            type="button"
            onClick={loadData}
            className="inline-flex items-center gap-2 bg-gray-800 hover:bg-gray-700 text-white px-4 py-2 rounded-lg text-sm transition"
          >
            <RefreshCw size={16} />
            Refresh
          </button>
        </div>
        {(!links.length || !accounts.length) && (
          <p className="text-xs text-gray-500 mt-3">
            Add at least one affiliate link and one Threads account before generating posts.
          </p>
        )}
      </form>

      {loading ? (
        <div className="text-center text-gray-400 py-8">Loading...</div>
      ) : posts.length === 0 ? (
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center">
          <Send className="mx-auto text-gray-600 mb-3" size={32} />
          <p className="text-gray-400">Belum ada posts. Generate konten dari affiliate link untuk mulai.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {posts.map((post) => {
            const status = statusConfig[post.status] || statusConfig.draft;
            const isBusy = busyPostID === post.id;

            return (
              <div key={post.id} className="bg-gray-900 rounded-xl border border-gray-800 p-5">
                <div className="flex items-start justify-between gap-3 mb-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${personaColors[post.persona] || 'bg-gray-700 text-gray-300'}`}>
                      {post.persona?.replace('_', ' ') || 'no persona'}
                    </span>
                    <span className="text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded">
                      {post.format || 'single'}
                    </span>
                    <span className="text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded">
                      {post.link_placement || 'direct'}
                    </span>
                  </div>
                  <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs ${status.color} ${status.bg}`}>
                    {status.icon}
                    {post.status}
                  </span>
                </div>

                <p className="text-sm text-gray-200 whitespace-pre-wrap leading-relaxed">
                  {post.content}
                </p>

                <div className="flex flex-col gap-3 mt-3 pt-3 border-t border-gray-800 sm:flex-row sm:items-center sm:justify-between">
                  <span className="text-xs text-gray-500">
                    Scheduled: {post.scheduled_at ? new Date(post.scheduled_at).toLocaleString('id-ID') : 'Not scheduled'}
                  </span>
                  <div className="flex gap-2">
                    {post.status === 'pending_review' && (
                      <button
                        onClick={() => approvePost(post.id)}
                        disabled={isBusy}
                        className="text-xs bg-indigo-600 hover:bg-indigo-700 text-white px-3 py-1.5 rounded transition disabled:opacity-50"
                      >
                        {isBusy ? 'Approving...' : 'Approve'}
                      </button>
                    )}
                    {post.status === 'approved' && (
                      <button
                        onClick={() => publishPost(post.id)}
                        disabled={isBusy}
                        className="text-xs bg-green-600 hover:bg-green-700 text-white px-3 py-1.5 rounded transition disabled:opacity-50"
                      >
                        {isBusy ? 'Queueing...' : 'Publish Now'}
                      </button>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
