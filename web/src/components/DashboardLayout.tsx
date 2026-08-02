import { Link, useNavigate, useLocation } from 'react-router-dom';
import { LayoutDashboard, User, LogOut } from 'lucide-react';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate();
  const location = useLocation();

  const handleLogout = () => {
    document.cookie = "vyala_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
    navigate('/login');
    window.location.reload(); 
  };

  const navLinkClass = (path: string) => 
    `flex items-center gap-2 px-4 py-2 rounded font-medium ${
      location.pathname === path ? 'bg-blue-50 text-blue-700' : 'text-gray-600 hover:bg-gray-100'
    }`;

  return (
    <div className="min-h-screen flex bg-gray-50">
      <aside className="w-64 bg-white border-r p-4 flex flex-col">
        <div className="mb-8 px-2 text-sm font-semibold text-slate-500">
          Navigation
        </div>
        <nav className="flex-1 space-y-2">
          <Link to="/dashboard" className={navLinkClass('/dashboard')}>
            <LayoutDashboard size={18} /> Repositories
          </Link>
          <Link to="/account" className={navLinkClass('/account')}>
            <User size={18} /> Account
          </Link>
        </nav>
        <button onClick={handleLogout} className="flex items-center gap-2 px-4 py-2 text-gray-600 hover:bg-gray-100 rounded font-medium">
          <LogOut size={18} /> Logout
        </button>
      </aside>
      <main className="flex-1 p-8 overflow-auto">
        {children}
      </main>
    </div>
  );
}