import { useEffect, useState } from 'react';
import { Server, Users, Activity, Clock } from 'lucide-react';

interface Stats {
  guilds: number;
  users: number;
  commands: number;
  uptime: string;
}

export default function DashboardHome() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await fetch('http://localhost:8080/api/stats');
        if (!response.ok) {
          throw new Error('Failed to fetch stats');
        }
        const data = await response.json();
        setStats(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchStats();

    // Refresh every 5 seconds for "real-time" feel until WebSocket is ready
    const interval = setInterval(fetchStats, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="p-8">
      <header className="mb-8">
        <h1 className="text-3xl font-bold text-white mb-2">Overview</h1>
        <p className="text-slate-400">Welcome to the JulesCord command center.</p>
      </header>

      {error && (
        <div className="bg-red-500/10 border border-red-500/50 text-red-400 px-4 py-3 rounded-lg mb-8">
          Error loading stats: {error}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Status Card */}
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700 shadow-sm relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
            <Activity className="w-24 h-24" />
          </div>
          <div className="flex items-center space-x-3 mb-4">
            <div className="w-10 h-10 rounded-lg bg-emerald-500/20 flex items-center justify-center">
              <Activity className="w-5 h-5 text-emerald-400" />
            </div>
            <h2 className="text-lg font-semibold text-slate-200">System Status</h2>
          </div>
          <div className="flex items-end space-x-2">
            <span className="text-3xl font-bold text-emerald-400">Online</span>
          </div>
          <div className="mt-4 text-sm text-slate-400 flex items-center">
            <span className="w-2 h-2 rounded-full bg-emerald-500 mr-2 animate-pulse"></span>
            All systems operational
          </div>
        </div>

        {/* Guilds Card */}
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700 shadow-sm relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
            <Server className="w-24 h-24" />
          </div>
          <div className="flex items-center space-x-3 mb-4">
            <div className="w-10 h-10 rounded-lg bg-indigo-500/20 flex items-center justify-center">
              <Server className="w-5 h-5 text-indigo-400" />
            </div>
            <h2 className="text-lg font-semibold text-slate-200">Servers</h2>
          </div>
          <div className="flex items-end space-x-2">
            <span className="text-3xl font-bold text-white">
              {loading ? '...' : stats?.guilds || 0}
            </span>
            <span className="text-sm text-slate-400 mb-1">Active</span>
          </div>
        </div>

        {/* Users Card */}
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700 shadow-sm relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
            <Users className="w-24 h-24" />
          </div>
          <div className="flex items-center space-x-3 mb-4">
            <div className="w-10 h-10 rounded-lg bg-cyan-500/20 flex items-center justify-center">
              <Users className="w-5 h-5 text-cyan-400" />
            </div>
            <h2 className="text-lg font-semibold text-slate-200">Users</h2>
          </div>
          <div className="flex items-end space-x-2">
            <span className="text-3xl font-bold text-white">
              {loading ? '...' : stats?.users || 0}
            </span>
            <span className="text-sm text-slate-400 mb-1">Tracked</span>
          </div>
        </div>

        {/* Commands Card */}
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700 shadow-sm relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
            <Clock className="w-24 h-24" />
          </div>
          <div className="flex items-center space-x-3 mb-4">
            <div className="w-10 h-10 rounded-lg bg-purple-500/20 flex items-center justify-center">
              <Clock className="w-5 h-5 text-purple-400" />
            </div>
            <h2 className="text-lg font-semibold text-slate-200">Commands</h2>
          </div>
          <div className="flex items-end space-x-2">
            <span className="text-3xl font-bold text-white">
              {loading ? '...' : stats?.commands || 0}
            </span>
            <span className="text-sm text-slate-400 mb-1">Executed</span>
          </div>
          <div className="mt-4 text-xs text-slate-400">
            Uptime: {loading ? '...' : stats?.uptime || 'N/A'}
          </div>
        </div>
      </div>
    </div>
  );
}
