import React, { useState, useEffect } from 'react';
import { apiClient } from '../api';
import { Coins, Building, TrendingUp } from 'lucide-react';

export default function Economy() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchEconomy = async () => {
      try {
        const res = await apiClient.get('/api/users');
        const economyUsers = res.data || [];

        // Sort by Coins + Bank combined, descending
        const sortedUsers = economyUsers
          .filter(u => u.coins > 0 || u.bank > 0)
          .sort((a, b) => (b.coins + b.bank) - (a.coins + a.bank))
          .slice(0, 50);

        setUsers(sortedUsers);
      } catch (err) {
        console.error("Failed to fetch economy data:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchEconomy();
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
        <h2 className="text-2xl font-bold text-white tracking-tight flex items-center">
          <TrendingUp className="w-6 h-6 mr-3 text-indigo-400" />
          Economy Leaderboard
        </h2>
        <div className="bg-indigo-600 px-4 py-2 rounded-lg text-sm font-medium text-white shadow-sm flex items-center">
          Top {users.length} Users
        </div>
      </div>

      <div className="bg-gray-800 rounded-xl border border-gray-700 shadow-sm overflow-hidden">
        {users.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            No economy data found. Start chatting to earn coins!
          </div>
        ) : (
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-gray-900/50 border-b border-gray-700">
                <th className="px-6 py-4 font-semibold text-gray-300 w-16">Rank</th>
                <th className="px-6 py-4 font-semibold text-gray-300">User</th>
                <th className="px-6 py-4 font-semibold text-gray-300">Net Worth</th>
                <th className="px-6 py-4 font-semibold text-gray-300">Wallet</th>
                <th className="px-6 py-4 font-semibold text-gray-300">Bank</th>
              </tr>
            </thead>
            <tbody>
              {users.map((user, idx) => (
                <tr
                  key={user.id}
                  className={`border-b border-gray-700/50 hover:bg-gray-700/30 transition-colors ${idx % 2 === 0 ? 'bg-gray-800' : 'bg-gray-800/80'}`}
                >
                  <td className="px-6 py-4">
                    <div className={`flex items-center justify-center w-8 h-8 rounded-full font-bold text-sm ${
                      idx === 0 ? 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/50' :
                      idx === 1 ? 'bg-gray-400/20 text-gray-300 border border-gray-400/50' :
                      idx === 2 ? 'bg-amber-700/20 text-amber-500 border border-amber-700/50' :
                      'bg-gray-700 text-gray-400'
                    }`}>
                      #{idx + 1}
                    </div>
                  </td>
                  <td className="px-6 py-4 text-white font-medium flex items-center">
                    {user.avatar_url ? (
                      <img src={user.avatar_url} alt="avatar" className="w-8 h-8 rounded-full mr-3 border border-gray-600" />
                    ) : (
                      <div className="w-8 h-8 rounded-full bg-indigo-500/20 text-indigo-400 flex items-center justify-center mr-3 font-sans font-bold text-xs border border-indigo-500/30">
                        {user.username.substring(0, 2).toUpperCase()}
                      </div>
                    )}
                    <div>
                      <div>{user.global_name || user.username}</div>
                      {user.global_name && <div className="text-xs text-gray-400">@{user.username}</div>}
                    </div>
                  </td>
                  <td className="px-6 py-4 text-indigo-400 font-bold text-sm">
                    {(user.coins + user.bank).toLocaleString()}
                  </td>
                  <td className="px-6 py-4 text-yellow-400 font-medium text-sm">
                     <div className="flex items-center">
                        <Coins className="w-4 h-4 mr-1"/>
                        {user.coins.toLocaleString()}
                     </div>
                  </td>
                  <td className="px-6 py-4 text-green-400 font-medium text-sm">
                     <div className="flex items-center">
                        <Building className="w-4 h-4 mr-1"/>
                        {user.bank.toLocaleString()}
                     </div>
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
