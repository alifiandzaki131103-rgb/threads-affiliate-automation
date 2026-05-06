import { useState, useEffect } from 'react';
import { Link2, FileText, MousePointerClick, TrendingUp } from 'lucide-react';
import api from '../api';

export default function Dashboard() {
  const [stats, setStats] = useState({ links: 0, posts: 0, clicks: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStats();
  }, []);

  const loadStats = async () => {
    try {
      const { data } = await api.get('/links');
      const totalClicks = (data.links || []).reduce((sum, l) => sum + (l.click_count || 0), 0);
      setStats({
        links: data.count || 0,
        posts: 0, // TODO: fetch from posts endpoint
        clicks: totalClicks,
      });
    } catch (err) {
      console.error('Failed to load stats:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h2 className="text-2xl font-bold text-white mb-6">Dashboard</h2>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <StatCard
          icon={<Link2 className="text-indigo-400" />}
          label="Affiliate Links"
          value={stats.links}
          loading={loading}
        />
        <StatCard
          icon={<FileText className="text-green-400" />}
          label="Posts Generated"
          value={stats.posts}
          loading={loading}
        />
        <StatCard
          icon={<MousePointerClick className="text-orange-400" />}
          label="Total Clicks"
          value={stats.clicks}
          loading={loading}
        />
      </div>

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
