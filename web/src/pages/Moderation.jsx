import React, { useState, useEffect } from 'react';
import { apiClient } from '../api';
import { Shield, Search, Filter } from 'lucide-react';

export default function Moderation() {
  const [actions, setActions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState('all');

  useEffect(() => {
    const fetchModActions = async () => {
      try {
        const res = await apiClient.get('http://localhost:8080/api/mod-actions');
        setActions(res.data || []);
      } catch (err) {
        console.error("Failed to fetch mod actions:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchModActions();
  }, []);

  const filteredActions = actions.filter(action => {
    const matchesSearch =
      action.target_username.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (action.target_global_name && action.target_global_name.toLowerCase().includes(searchTerm.toLowerCase())) ||
      action.target_id.includes(searchTerm) ||
      action.mod_username.toLowerCase().includes(searchTerm.toLowerCase()) ||
      action.reason.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesFilter = filterType === 'all' || action.action.toLowerCase() === filterType.toLowerCase();

    return matchesSearch && matchesFilter;
  });

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500"></div>
      </div>
    );
  }

  const getActionColor = (actionType) => {
      switch(actionType.toLowerCase()) {
          case 'ban': return 'bg-red-500/20 text-red-400 border border-red-500/30';
          case 'kick': return 'bg-orange-500/20 text-orange-400 border border-orange-500/30';
          case 'warn': return 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/30';
          case 'purge': return 'bg-blue-500/20 text-blue-400 border border-blue-500/30';
          default: return 'bg-gray-500/20 text-gray-400 border border-gray-500/30';
      }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-white tracking-tight">Moderation Logs</h2>
        <div className="bg-indigo-600 px-4 py-2 rounded-lg text-sm font-medium text-white shadow-sm flex items-center">
          <Shield className="w-4 h-4 mr-2" />
          {actions.length} Actions
        </div>
      </div>

      <div className="flex flex-col sm:flex-row gap-4">
          <div className="bg-gray-800 p-4 rounded-xl border border-gray-700 shadow-sm flex items-center flex-1">
            <Search className="w-5 h-5 text-gray-400 mr-3" />
            <input
              type="text"
              placeholder="Search by user, reason, or ID..."
              className="bg-transparent border-none text-white focus:outline-none w-full placeholder-gray-500"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>

          <div className="bg-gray-800 p-4 rounded-xl border border-gray-700 shadow-sm flex items-center">
            <Filter className="w-5 h-5 text-gray-400 mr-3" />
            <select
                className="bg-transparent text-white border-none focus:outline-none focus:ring-0 appearance-none pr-8 cursor-pointer"
                value={filterType}
                onChange={(e) => setFilterType(e.target.value)}
            >
                <option value="all" className="bg-gray-800">All Actions</option>
                <option value="warn" className="bg-gray-800">Warnings</option>
                <option value="kick" className="bg-gray-800">Kicks</option>
                <option value="ban" className="bg-gray-800">Bans</option>
                <option value="purge" className="bg-gray-800">Purges</option>
            </select>
          </div>
      </div>

      <div className="bg-gray-800 rounded-xl border border-gray-700 shadow-sm overflow-hidden">
        {filteredActions.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            No moderation actions found.
          </div>
        ) : (
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-gray-900/50 border-b border-gray-700">
                <th className="px-6 py-4 font-semibold text-gray-300">Action</th>
                <th className="px-6 py-4 font-semibold text-gray-300">Target User</th>
                <th className="px-6 py-4 font-semibold text-gray-300">Moderator</th>
                <th className="px-6 py-4 font-semibold text-gray-300">Reason</th>
                <th className="px-6 py-4 font-semibold text-gray-300">Date</th>
              </tr>
            </thead>
            <tbody>
              {filteredActions.map((action, idx) => (
                <tr
                  key={action.id}
                  className={`border-b border-gray-700/50 hover:bg-gray-700/30 transition-colors ${idx % 2 === 0 ? 'bg-gray-800' : 'bg-gray-800/80'}`}
                >
                  <td className="px-6 py-4 text-white font-medium">
                     <span className={`px-2.5 py-1 rounded-md text-xs font-bold uppercase tracking-wider ${getActionColor(action.action)}`}>
                        {action.action}
                     </span>
                  </td>
                  <td className="px-6 py-4 text-gray-300 text-sm">
                      <div className="flex items-center">
                           <div className="font-medium text-white">{action.target_global_name || action.target_username}</div>
                           <div className="text-xs text-gray-500 ml-2 font-mono">{action.target_id}</div>
                      </div>
                  </td>
                  <td className="px-6 py-4 text-gray-300 text-sm">
                      <div className="flex items-center">
                           <div className="font-medium text-gray-400">{action.mod_global_name || action.mod_username}</div>
                      </div>
                  </td>
                  <td className="px-6 py-4 text-gray-400 text-sm max-w-xs truncate" title={action.reason}>
                      {action.reason}
                  </td>
                  <td className="px-6 py-4 text-gray-500 font-mono text-xs">
                    {new Date(action.created_at).toLocaleString(undefined, {
                        year: 'numeric', month: 'short', day: 'numeric',
                        hour: '2-digit', minute: '2-digit'
                    })}
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
