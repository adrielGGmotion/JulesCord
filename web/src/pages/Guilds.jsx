import React, { useState, useEffect } from 'react';
import { apiClient } from '../api';
import { Server, Calendar, Settings } from 'lucide-react';
import { Link } from 'react-router-dom';

export default function Guilds() {
  const [guilds, setGuilds] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchGuilds = async () => {
      try {
        const res = await apiClient.get('/api/guilds');
        setGuilds(res.data || []);
      } catch (err) {
        console.error("Failed to fetch guilds:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchGuilds();
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-white tracking-tight">Connected Servers</h2>
        <div className="bg-indigo-600 px-4 py-2 rounded-lg text-sm font-medium text-white shadow-sm flex items-center">
          <Server className="w-4 h-4 mr-2" />
          {guilds.length} Total
        </div>
      </div>

      <div className="bg-gray-800 rounded-xl border border-gray-700 shadow-sm overflow-hidden">
        {guilds.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            No guilds connected yet.
          </div>
        ) : (
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-gray-900/50 border-b border-gray-700">
                <th className="px-6 py-4 font-semibold text-gray-300">Server ID</th>
                <th className="px-6 py-4 font-semibold text-gray-300">Joined Date</th>
                <th className="px-6 py-4 font-semibold text-gray-300 text-right">Action</th>
              </tr>
            </thead>
            <tbody>
              {guilds.map((guild, idx) => (
                <tr
                  key={guild.id}
                  className={`border-b border-gray-700/50 hover:bg-gray-700/30 transition-colors ${idx % 2 === 0 ? 'bg-gray-800' : 'bg-gray-800/80'}`}
                >
                  <td className="px-6 py-4 text-gray-300 font-mono text-sm flex items-center">
                    <div className="w-8 h-8 rounded-full bg-indigo-500/20 text-indigo-400 flex items-center justify-center mr-3 font-sans font-bold text-xs">
                      {guild.id.substring(0, 2)}
                    </div>
                    {guild.id}
                  </td>
                  <td className="px-6 py-4 text-gray-400 text-sm">
                    <div className="flex items-center">
                      <Calendar className="w-4 h-4 mr-2 text-gray-500" />
                      {new Date(guild.joined_at).toLocaleDateString(undefined, {
                        year: 'numeric',
                        month: 'short',
                        day: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit'
                      })}
                    </div>
                  </td>
                  <td className="px-6 py-4 text-right">
                    <Link
                      to={`/guilds/${guild.id}/settings`}
                      className="inline-flex items-center justify-center p-2 rounded-lg bg-gray-700/50 hover:bg-indigo-600 hover:text-white text-gray-400 transition-all border border-gray-600/50 hover:border-indigo-500 shadow-sm group"
                      title="Manage Settings"
                    >
                      <Settings className="w-4 h-4 group-hover:rotate-45 transition-transform duration-300" />
                    </Link>
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
