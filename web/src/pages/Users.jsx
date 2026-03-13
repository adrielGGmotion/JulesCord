import React, { useState, useEffect } from 'react';
import { apiClient } from '../api';
import { Users as UsersIcon, Search, Award } from 'lucide-react';

export default function Users() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    const fetchUsers = async () => {
      try {
        const res = await apiClient.get('http://localhost:8080/api/users');
        setUsers(res.data || []);
      } catch (err) {
        console.error("Failed to fetch users:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchUsers();
  }, []);

  const filteredUsers = users.filter(user =>
    user.username.toLowerCase().includes(searchTerm.toLowerCase()) ||
    (user.global_name && user.global_name.toLowerCase().includes(searchTerm.toLowerCase())) ||
    user.id.includes(searchTerm)
  );

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
        <h2 className="text-2xl font-bold text-white tracking-tight">Users</h2>
        <div className="bg-indigo-600 px-4 py-2 rounded-lg text-sm font-medium text-white shadow-sm flex items-center">
          <UsersIcon className="w-4 h-4 mr-2" />
          {users.length} Total
        </div>
      </div>

      <div className="bg-gray-800 p-4 rounded-xl border border-gray-700 shadow-sm flex items-center">
        <Search className="w-5 h-5 text-gray-400 mr-3" />
        <input
          type="text"
          placeholder="Search users by name or ID..."
          className="bg-transparent border-none text-white focus:outline-none w-full placeholder-gray-500"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
      </div>

      <div className="bg-gray-800 rounded-xl border border-gray-700 shadow-sm overflow-hidden">
        {filteredUsers.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            No users found.
          </div>
        ) : (
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-gray-900/50 border-b border-gray-700">
                <th className="px-6 py-4 font-semibold text-gray-300">User</th>
                <th className="px-6 py-4 font-semibold text-gray-300">ID</th>
                <th className="px-6 py-4 font-semibold text-gray-300">Total XP</th>
                <th className="px-6 py-4 font-semibold text-gray-300">Max Level</th>
              </tr>
            </thead>
            <tbody>
              {filteredUsers.map((user, idx) => (
                <tr
                  key={user.id}
                  className={`border-b border-gray-700/50 hover:bg-gray-700/30 transition-colors ${idx % 2 === 0 ? 'bg-gray-800' : 'bg-gray-800/80'}`}
                >
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
                  <td className="px-6 py-4 text-gray-400 font-mono text-sm">
                    {user.id}
                  </td>
                  <td className="px-6 py-4 text-gray-300 font-mono text-sm">
                    {user.total_xp.toLocaleString()}
                  </td>
                  <td className="px-6 py-4 text-indigo-400 font-bold text-sm">
                     <div className="flex items-center">
                        <Award className="w-4 h-4 mr-1"/>
                        {user.max_level}
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
