import { Outlet, Link, useNavigate } from 'react-router-dom';
import { LayoutDashboard, Link2, FileText, LogOut } from 'lucide-react';

export default function Layout() {
  const navigate = useNavigate();

  const logout = () => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    navigate('/login');
  };

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100 flex">
      {/* Sidebar */}
      <aside className="w-64 bg-gray-900 border-r border-gray-800 p-4 flex flex-col">
        <div className="mb-8">
          <h1 className="text-xl font-bold text-indigo-400">Threads Affiliate</h1>
          <p className="text-xs text-gray-500 mt-1">AI-Powered Automation</p>
        </div>

        <nav className="flex-1 space-y-1">
          <NavItem to="/" icon={<LayoutDashboard size={18} />} label="Dashboard" />
          <NavItem to="/links" icon={<Link2 size={18} />} label="Links" />
          <NavItem to="/posts" icon={<FileText size={18} />} label="Posts" />
        </nav>

        <button
          onClick={logout}
          className="flex items-center gap-2 px-3 py-2 text-sm text-gray-400 hover:text-red-400 hover:bg-gray-800 rounded-lg transition"
        >
          <LogOut size={18} />
          Logout
        </button>
      </aside>

      {/* Main content */}
      <main className="flex-1 p-6 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}

function NavItem({ to, icon, label }) {
  return (
    <Link
      to={to}
      className="flex items-center gap-2 px-3 py-2 text-sm text-gray-300 hover:text-white hover:bg-gray-800 rounded-lg transition"
    >
      {icon}
      {label}
    </Link>
  );
}
