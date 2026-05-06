import { useState, useEffect } from 'react';
import { Link2, FileText, MousePointerClick, TrendingUp, BarChart3 } from 'lucide-react';
import api from '../api';

export default function Dashboard() {
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);

  async function loadStats() {
    try {
      const { data } = await api.get('/dashboard');
      setStats(data);
    } catch (err) {
      console.error('Failed to load stats:', err);
      setStats({ total_links: 0, total_clicks: 0, total_posts: 0, published_posts: 0, pending_posts: 0, top_links: [] });
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadStats();
  }, []);

  return (
    <div>
      <h2 className="text-2xl font-bold text-white mb-6">Dashboard</h2>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <StatCard
          icon={<Link2 className="text-indigo-400" />}
          label="Affiliate Links"
          value={stats?.total_links || 0}
          loading={loading}
        />
        <StatCard
          icon={<MousePointerClick className="text-orange-400" />}
          label="Total Clicks"
          value={stats?.total_clicks || 0}
          loading={loading}
        />
        <StatCard
          icon={<FileText className="text-green-400" />}
          label="Published Posts"
          value={stats?.published_posts || 0}
          loading={loading}
        />
        <StatCard
          icon={<BarChart3 className="text-purple-400" />}
          label="Pending Posts"
          value={stats?.pending_posts || 0}
          loading={loading}
        />
      </div>

      {/* Top Links */}
      {stats?.top_links?.length > 0 && (
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-6 mb-6">
          <h3 className="text-lg font-semibold text-white mb-4">🔥 Top Performing Links</h3>
          <div className="space-y-3">
            {stats.top_links.map((link, i) => (
              <div key={link.id} className="flex items-center justify-between p-3 bg-gray-800 rounded-lg">
                <div className="flex items-center gap-3">
                  <span className="text-lg font-bold text-indigo-400">#{i + 1}</span>
                  <div>
                    <p className="text-sm font-medium text-white truncate max-w-[300px]">{link.product_name}</p>
                    <p className="text-xs text-gray-400">{link.platform} • /s/{link.short_slug}</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-lg font-bold text-orange-400">{link.click_count}</p>
                  <p className="text-xs text-gray-500">clicks</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Quick Actions */}
      <div className="bg-gray-900 rounded-xl border border-gray-800 p-6">
        <h3 className="text-lg font-semibold text-white mb-4">Quick Actions</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <QuickAction
            href="/links"
            icon={<Link2 size={20} />}
            title="Add Affiliate Links"
            desc="Paste links dari Shopee/TikTok"
          />
          <QuickAction
            href="/posts"
            icon={<FileText size={20} />}
            title="View Posts"
            desc="Lihat konten yang di-generate AI"
          />
        </div>
      </div>

      {/* Info */}
      <div className="mt-6 bg-indigo-950/30 border border-indigo-800/50 rounded-xl p-4">
        <div className="flex items-start gap-3">
          <TrendingUp className="text-indigo-400 mt-0.5" size={20} />
          <div>
            <p className="text-sm text-indigo-200 font-medium">AI Self-Learning Active</p>
            <p className="text-xs text-indigo-300/70 mt-1">
              AI mengoptimasi konten berdasarkan clicks & engagement. Report mingguan setiap Senin.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

function StatCard({ icon, label, value, loading }) {
  return (
    <div className="bg-gray-900 rounded-xl border border-gray-800 p-4">
      <div className="flex items-center gap-3">
        {icon}
        <div>
          <p className="text-xs text-gray-400">{label}</p>
          <p className="text-2xl font-bold text-white">
            {loading ? '...' : value}
          </p>
        </div>
      </div>
    </div>
  );
}

function QuickAction({ href, icon, title, desc }) {
  return (
    <a
      href={href}
      className="flex items-center gap-3 p-3 bg-gray-800 hover:bg-gray-750 rounded-lg border border-gray-700 hover:border-gray-600 transition"
    >
      <div className="text-indigo-400">{icon}</div>
      <div>
        <p className="text-sm font-medium text-white">{title}</p>
        <p className="text-xs text-gray-400">{desc}</p>
      </div>
    </a>
  );
}
