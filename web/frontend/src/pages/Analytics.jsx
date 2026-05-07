import { useState, useEffect, useMemo } from 'react';
import { BarChart3, TrendingUp, Clock, Brain } from 'lucide-react';
import api from '../api';

export default function Analytics() {
  const [period, setPeriod] = useState('7d');
  const [analytics, setAnalytics] = useState(null);
  const [insights, setInsights] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchData();
  }, [period]);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [analyticsRes, insightsRes] = await Promise.all([
        api.get(`/analytics?period=${period}`),
        api.get('/insights'),
      ]);
      setAnalytics(analyticsRes.data);
      setInsights(insightsRes.data);
    } catch (err) {
      console.error('Failed to fetch analytics:', err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <BarChart3 className="text-indigo-400" size={28} />
            Analytics
          </h1>
          <p className="text-gray-400 text-sm mt-1">Performance insights and recommendations</p>
        </div>
        <PeriodSelector period={period} setPeriod={setPeriod} />
      </div>

      {/* Summary Cards */}
      {analytics && insights?.current_week && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <SummaryCard
            label="Total Clicks"
            value={analytics.total_clicks}
            icon={<TrendingUp size={20} />}
            color="indigo"
          />
          <SummaryCard
            label="Avg Daily Clicks"
            value={analytics.avg_daily_clicks?.toFixed(1)}
            icon={<BarChart3 size={20} />}
            color="purple"
          />
          <SummaryCard
            label="Posts Published"
            value={insights.current_week.posts_published}
            icon={<Clock size={20} />}
            color="orange"
          />
          <SummaryCard
            label="Avg Clicks/Post"
            value={insights.current_week.avg_clicks_per_post?.toFixed(1)}
            icon={<Brain size={20} />}
            color="emerald"
          />
        </div>
      )}

      {/* Line Chart - Clicks by Day */}
      {analytics?.clicks_by_day && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <TrendingUp size={20} className="text-indigo-400" />
            Clicks Over Time
          </h2>
          <LineChart data={analytics.clicks_by_day} />
        </div>
      )}

      {/* Performance Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Persona Performance */}
        {insights?.persona_performance && (
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
            <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
              <BarChart3 size={20} className="text-purple-400" />
              Persona Performance
            </h2>
            <HorizontalBarChart
              data={insights.persona_performance.map(p => ({
                label: p.persona.replace(/_/g, ' '),
                value: p.clicks,
                secondary: `${p.posts} posts · weight ${p.weight.toFixed(1)}`,
              }))}
              color="purple"
            />
          </div>
        )}

        {/* Format Performance */}
        {insights?.format_performance && (
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
            <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
              <BarChart3 size={20} className="text-orange-400" />
              Format Performance
            </h2>
            <HorizontalBarChart
              data={insights.format_performance.map(f => ({
                label: f.format,
                value: f.clicks,
                secondary: `${f.posts} posts · weight ${f.weight.toFixed(1)}`,
              }))}
              color="orange"
            />
          </div>
        )}
      </div>

      {/* Best Posting Hours */}
      {insights?.best_posting_hours && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <Clock size={20} className="text-indigo-400" />
            Best Posting Hours
          </h2>
          <HourHeatmap data={insights.best_posting_hours} />
        </div>
      )}

      {/* AI Recommendations */}
      {insights?.recommendations && insights.recommendations.length > 0 && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
            <Brain size={20} className="text-emerald-400" />
            AI Recommendations
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {insights.recommendations.map((rec, i) => (
              <RecommendationCard key={i} text={rec} index={i} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function PeriodSelector({ period, setPeriod }) {
  const periods = [
    { value: '1d', label: '24h' },
    { value: '7d', label: '7 days' },
    { value: '30d', label: '30 days' },
  ];

  return (
    <div className="flex bg-gray-800 rounded-lg p-1">
      {periods.map(p => (
        <button
          key={p.value}
          onClick={() => setPeriod(p.value)}
          className={`px-4 py-1.5 text-sm font-medium rounded-md transition ${
            period === p.value
              ? 'bg-indigo-600 text-white shadow'
              : 'text-gray-400 hover:text-white'
          }`}
        >
          {p.label}
        </button>
      ))}
    </div>
  );
}

function SummaryCard({ label, value, icon, color }) {
  const colorMap = {
    indigo: 'from-indigo-500/20 to-indigo-600/5 border-indigo-500/30 text-indigo-400',
    purple: 'from-purple-500/20 to-purple-600/5 border-purple-500/30 text-purple-400',
    orange: 'from-orange-500/20 to-orange-600/5 border-orange-500/30 text-orange-400',
    emerald: 'from-emerald-500/20 to-emerald-600/5 border-emerald-500/30 text-emerald-400',
  };

  const iconColorMap = {
    indigo: 'text-indigo-400',
    purple: 'text-purple-400',
    orange: 'text-orange-400',
    emerald: 'text-emerald-400',
  };

  return (
    <div className={`bg-gradient-to-br ${colorMap[color]} border rounded-xl p-4`}>
      <div className="flex items-center justify-between mb-2">
        <span className="text-gray-400 text-sm">{label}</span>
        <span className={iconColorMap[color]}>{icon}</span>
      </div>
      <p className="text-2xl font-bold text-white">{value ?? '—'}</p>
    </div>
  );
}

function LineChart({ data }) {
  const width = 700;
  const height = 200;
  const padding = { top: 20, right: 20, bottom: 40, left: 50 };

  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;

  const maxClicks = Math.max(...data.map(d => d.clicks), 1);
  const minClicks = 0;

  const points = data.map((d, i) => ({
    x: padding.left + (i / Math.max(data.length - 1, 1)) * chartWidth,
    y: padding.top + chartHeight - ((d.clicks - minClicks) / (maxClicks - minClicks || 1)) * chartHeight,
    date: d.date,
    clicks: d.clicks,
  }));

  const pathD = points.length > 0
    ? `M ${points.map(p => `${p.x},${p.y}`).join(' L ')}`
    : '';

  const areaD = points.length > 0
    ? `${pathD} L ${points[points.length - 1].x},${padding.top + chartHeight} L ${points[0].x},${padding.top + chartHeight} Z`
    : '';

  // Y-axis labels
  const yLabels = [0, Math.round(maxClicks / 2), maxClicks];

  return (
    <div className="w-full overflow-x-auto">
      <svg viewBox={`0 0 ${width} ${height}`} className="w-full h-auto min-w-[500px]">
        {/* Grid lines */}
        {yLabels.map((val, i) => {
          const y = padding.top + chartHeight - (val / maxClicks) * chartHeight;
          return (
            <g key={i}>
              <line
                x1={padding.left}
                y1={y}
                x2={width - padding.right}
                y2={y}
                stroke="#374151"
                strokeDasharray="4,4"
              />
              <text x={padding.left - 10} y={y + 4} textAnchor="end" fill="#9CA3AF" fontSize="11">
                {val}
              </text>
            </g>
          );
        })}

        {/* Area fill */}
        <defs>
          <linearGradient id="areaGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#6366F1" stopOpacity="0.3" />
            <stop offset="100%" stopColor="#6366F1" stopOpacity="0" />
          </linearGradient>
        </defs>
        <path d={areaD} fill="url(#areaGradient)" />

        {/* Line */}
        <path d={pathD} fill="none" stroke="#6366F1" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />

        {/* Data points */}
        {points.map((p, i) => (
          <circle key={i} cx={p.x} cy={p.y} r="4" fill="#6366F1" stroke="#1F2937" strokeWidth="2" />
        ))}

        {/* X-axis labels */}
        {points.map((p, i) => {
          // Show every nth label to avoid crowding
          const showEvery = data.length > 14 ? 4 : data.length > 7 ? 2 : 1;
          if (i % showEvery !== 0 && i !== data.length - 1) return null;
          const dateStr = p.date.slice(5); // MM-DD
          return (
            <text key={i} x={p.x} y={height - 8} textAnchor="middle" fill="#9CA3AF" fontSize="10">
              {dateStr}
            </text>
          );
        })}
      </svg>
    </div>
  );
}

function HorizontalBarChart({ data, color }) {
  const maxValue = Math.max(...data.map(d => d.value), 1);

  const barColorMap = {
    purple: 'bg-purple-500',
    orange: 'bg-orange-500',
    indigo: 'bg-indigo-500',
  };

  const bgColorMap = {
    purple: 'bg-purple-500/10',
    orange: 'bg-orange-500/10',
    indigo: 'bg-indigo-500/10',
  };

  return (
    <div className="space-y-3">
      {data.map((item, i) => (
        <div key={i}>
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-gray-200 capitalize font-medium">{item.label}</span>
            <span className="text-sm text-gray-400">{item.value} clicks</span>
          </div>
          <div className={`w-full h-3 rounded-full ${bgColorMap[color]}`}>
            <div
              className={`h-3 rounded-full ${barColorMap[color]} transition-all duration-500`}
              style={{ width: `${(item.value / maxValue) * 100}%` }}
            ></div>
          </div>
          {item.secondary && (
            <p className="text-xs text-gray-500 mt-0.5">{item.secondary}</p>
          )}
        </div>
      ))}
    </div>
  );
}

function HourHeatmap({ data }) {
  // Build a map of hour -> data
  const hourMap = {};
  data.forEach(d => {
    hourMap[d.hour] = d;
  });

  const maxClicks = Math.max(...data.map(d => d.clicks), 1);

  const getIntensity = (hour) => {
    const entry = hourMap[hour];
    if (!entry) return 0;
    return entry.clicks / maxClicks;
  };

  const getColor = (intensity) => {
    if (intensity === 0) return 'bg-gray-800';
    if (intensity < 0.25) return 'bg-indigo-900/50';
    if (intensity < 0.5) return 'bg-indigo-700/60';
    if (intensity < 0.75) return 'bg-indigo-500/70';
    return 'bg-indigo-400';
  };

  const formatHour = (h) => {
    if (h === 0) return '12am';
    if (h < 12) return `${h}am`;
    if (h === 12) return '12pm';
    return `${h - 12}pm`;
  };

  return (
    <div>
      <div className="grid grid-cols-12 gap-2 mb-2">
        {Array.from({ length: 24 }, (_, hour) => {
          const intensity = getIntensity(hour);
          const entry = hourMap[hour];
          return (
            <div
              key={hour}
              className={`relative group rounded-lg p-2 text-center ${getColor(intensity)} border border-gray-700/50 transition-all hover:scale-105 hover:border-indigo-500/50`}
            >
              <p className="text-xs text-gray-400 font-medium">{formatHour(hour)}</p>
              <p className="text-sm font-bold text-white mt-1">
                {entry ? entry.clicks : '—'}
              </p>
              {entry && (
                <p className="text-[10px] text-gray-500">w:{entry.weight.toFixed(1)}</p>
              )}
            </div>
          );
        })}
      </div>
      <div className="flex items-center gap-2 mt-4 justify-end">
        <span className="text-xs text-gray-500">Less</span>
        <div className="flex gap-1">
          <div className="w-4 h-4 rounded bg-gray-800 border border-gray-700"></div>
          <div className="w-4 h-4 rounded bg-indigo-900/50 border border-gray-700"></div>
          <div className="w-4 h-4 rounded bg-indigo-700/60 border border-gray-700"></div>
          <div className="w-4 h-4 rounded bg-indigo-500/70 border border-gray-700"></div>
          <div className="w-4 h-4 rounded bg-indigo-400 border border-gray-700"></div>
        </div>
        <span className="text-xs text-gray-500">More</span>
      </div>
    </div>
  );
}

function RecommendationCard({ text, index }) {
  const colors = [
    'border-indigo-500/30 bg-indigo-500/5',
    'border-purple-500/30 bg-purple-500/5',
    'border-orange-500/30 bg-orange-500/5',
    'border-emerald-500/30 bg-emerald-500/5',
    'border-cyan-500/30 bg-cyan-500/5',
  ];

  const iconColors = [
    'text-indigo-400',
    'text-purple-400',
    'text-orange-400',
    'text-emerald-400',
    'text-cyan-400',
  ];

  const colorClass = colors[index % colors.length];
  const iconColor = iconColors[index % iconColors.length];

  return (
    <div className={`border rounded-xl p-4 ${colorClass} transition hover:scale-[1.02]`}>
      <div className="flex items-start gap-3">
        <Brain size={18} className={`${iconColor} mt-0.5 flex-shrink-0`} />
        <p className="text-sm text-gray-200 leading-relaxed">{text}</p>
      </div>
    </div>
  );
}
