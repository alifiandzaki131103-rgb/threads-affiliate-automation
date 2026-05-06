import { useState, useEffect } from 'react';
import { Clock, CheckCircle, XCircle, Send } from 'lucide-react';

export default function Posts() {
  const [posts, setPosts] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // TODO: fetch from /api/posts when endpoint is ready
    setLoading(false);
    setPosts([
      // Sample data for UI preview
      {
        id: '1',
        content: 'Jujur gue skeptis awalnya sama serum vit C yang lagi rame. Harga 89rb, review 4.9 dari 15rb orang...',
        persona: 'honest_friend',
        format: 'single',
        status: 'published',
        scheduled_at: '2026-05-07T10:00:00Z',
        link_placement: 'direct',
      },
      {
        id: '2',
        content: 'Unpopular opinion: earphone 50rb di Shopee itu 80% sama kualitasnya dengan yang 500rb...',
        persona: 'hot_take',
        format: 'hot_take',
        status: 'approved',
        scheduled_at: '2026-05-07T14:30:00Z',
        link_placement: 'reply_drop',
      },
      {
        id: '3',
        content: 'Morning routine update: sekarang nambah serum vitamin C di step 3...',
        persona: 'lifestyle_sharer',
        format: 'single',
        status: 'pending_review',
        scheduled_at: '2026-05-07T18:00:00Z',
        link_placement: 'bio',
      },
    ]);
  }, []);

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

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-white">Posts</h2>
        <div className="flex gap-2">
          <span className="text-sm text-gray-400 bg-gray-800 px-3 py-1 rounded-lg">
            {posts.filter(p => p.status === 'published').length} published
          </span>
          <span className="text-sm text-gray-400 bg-gray-800 px-3 py-1 rounded-lg">
            {posts.filter(p => p.status === 'approved').length} scheduled
          </span>
        </div>
      </div>

      {loading ? (
        <div className="text-center text-gray-400 py-8">Loading...</div>
      ) : posts.length === 0 ? (
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center">
          <Send className="mx-auto text-gray-600 mb-3" size={32} />
          <p className="text-gray-400">Belum ada posts. Tambahkan affiliate links dulu, AI akan generate konten otomatis.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {posts.map((post) => {
            const status = statusConfig[post.status] || statusConfig.draft;
            return (
              <div key={post.id} className="bg-gray-900 rounded-xl border border-gray-800 p-5">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${personaColors[post.persona] || 'bg-gray-700 text-gray-300'}`}>
                      {post.persona?.replace('_', ' ')}
                    </span>
                    <span className="text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded">
                      {post.format}
                    </span>
                    <span className="text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded">
                      {post.link_placement}
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

                <div className="flex items-center justify-between mt-3 pt-3 border-t border-gray-800">
                  <span className="text-xs text-gray-500">
                    Scheduled: {new Date(post.scheduled_at).toLocaleString('id-ID')}
                  </span>
                  {post.status === 'pending_review' && (
                    <button className="text-xs bg-indigo-600 hover:bg-indigo-700 text-white px-3 py-1 rounded transition">
                      Approve
                    </button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
