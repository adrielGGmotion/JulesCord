
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import { Home, Server, Users, Shield, Settings, Activity } from 'lucide-react';
import DashboardHome from './pages/DashboardHome';
import GuildsPage from './pages/GuildsPage';

function App() {
  return (
    <Router>
      <div className="flex h-screen bg-slate-900 text-slate-50 overflow-hidden">
        {/* Sidebar */}
        <aside className="w-64 bg-slate-800 border-r border-slate-700 flex flex-col">
          <div className="p-6 border-b border-slate-700 flex items-center space-x-3">
            <div className="w-10 h-10 rounded-full bg-indigo-500 flex items-center justify-center">
              <Activity className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-indigo-400 to-cyan-400">
                JulesCord
              </h1>
              <p className="text-xs text-slate-400">Dashboard</p>
            </div>
          </div>

          <nav className="flex-1 p-4 space-y-2">
            <Link to="/" className="flex items-center space-x-3 px-4 py-3 rounded-lg bg-indigo-500/10 text-indigo-400 hover:bg-indigo-500/20 transition-colors">
              <Home className="w-5 h-5" />
              <span className="font-medium">Overview</span>
            </Link>
            <Link to="/guilds" className="flex items-center space-x-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-700 hover:text-white transition-colors">
              <Server className="w-5 h-5" />
              <span className="font-medium">Servers</span>
            </Link>
            <Link to="#" className="flex items-center space-x-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-700 hover:text-white transition-colors opacity-50 cursor-not-allowed" title="Coming Soon">
              <Users className="w-5 h-5" />
              <span className="font-medium">Users</span>
            </Link>
            <Link to="#" className="flex items-center space-x-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-700 hover:text-white transition-colors opacity-50 cursor-not-allowed" title="Coming Soon">
              <Shield className="w-5 h-5" />
              <span className="font-medium">Moderation</span>
            </Link>
            <Link to="#" className="flex items-center space-x-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-700 hover:text-white transition-colors opacity-50 cursor-not-allowed" title="Coming Soon">
              <Settings className="w-5 h-5" />
              <span className="font-medium">Settings</span>
            </Link>
          </nav>

          <div className="p-4 border-t border-slate-700">
            <div className="flex items-center space-x-2 text-sm text-slate-400">
              <div className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></div>
              <span>Bot Online</span>
            </div>
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1 overflow-y-auto">
          <Routes>
            <Route path="/" element={<DashboardHome />} />
            <Route path="/guilds" element={<GuildsPage />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;
