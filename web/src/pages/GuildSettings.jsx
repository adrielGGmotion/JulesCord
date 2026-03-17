import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { apiClient } from '../api';
import { Settings, Save, ArrowLeft, CheckCircle2, AlertCircle } from 'lucide-react';

export default function GuildSettings() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState(null);

  const [config, setConfig] = useState({
    prefix: '!',
    log_channel_id: '',
    welcome_channel_id: '',
    mod_role_id: '',
    auto_role_id: '',
    counting_channel_id: '',
    suggestion_channel_id: ''
  });

  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const res = await apiClient.get(`/api/guilds/${id}/config`);
        if (res.data) {
          setConfig({
            prefix: res.data.prefix || '!',
            log_channel_id: res.data.log_channel_id || '',
            welcome_channel_id: res.data.welcome_channel_id || '',
            mod_role_id: res.data.mod_role_id || '',
            auto_role_id: res.data.auto_role_id || '',
            counting_channel_id: res.data.counting_channel_id || '',
            suggestion_channel_id: res.data.suggestion_channel_id || ''
          });
        }
      } catch (err) {
        console.error("Failed to fetch guild config:", err);
        setMessage({ type: 'error', text: 'Failed to load settings.' });
      } finally {
        setLoading(false);
      }
    };
    fetchConfig();
  }, [id]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setConfig(prev => ({
      ...prev,
      [name]: value
    }));
  };

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    setMessage(null);

    // Prepare payload, converting empty strings to null for optional IDs
    const payload = {
      prefix: config.prefix,
      log_channel_id: config.log_channel_id,
      welcome_channel_id: config.welcome_channel_id,
      mod_role_id: config.mod_role_id,
      auto_role_id: config.auto_role_id,
      counting_channel_id: config.counting_channel_id,
      suggestion_channel_id: config.suggestion_channel_id
    };

    try {
      await apiClient.patch(`/api/guilds/${id}/config`, payload);
      setMessage({ type: 'success', text: 'Settings saved successfully.' });
    } catch (err) {
      console.error("Failed to save guild config:", err);
      setMessage({ type: 'error', text: 'Failed to save settings.' });
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500"></div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="flex items-center space-x-4">
        <button
          onClick={() => navigate('/guilds')}
          className="p-2 hover:bg-gray-800 rounded-lg text-gray-400 hover:text-white transition-colors"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
        <div className="flex items-center">
          <Settings className="w-6 h-6 mr-3 text-indigo-400" />
          <h2 className="text-2xl font-bold text-white tracking-tight">
            Server Settings <span className="text-gray-500 text-lg ml-2 font-mono">({id})</span>
          </h2>
        </div>
      </div>

      {message && (
        <div className={`p-4 rounded-lg flex items-center ${
          message.type === 'success' ? 'bg-green-500/10 border border-green-500/20 text-green-400' : 'bg-red-500/10 border border-red-500/20 text-red-400'
        }`}>
          {message.type === 'success' ? <CheckCircle2 className="w-5 h-5 mr-3" /> : <AlertCircle className="w-5 h-5 mr-3" />}
          {message.text}
        </div>
      )}

      <form onSubmit={handleSave} className="bg-gray-800 rounded-xl border border-gray-700 shadow-sm p-6 space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">Command Prefix</label>
            <input
              type="text"
              name="prefix"
              value={config.prefix}
              onChange={handleChange}
              placeholder="!"
              className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors"
            />
            <p className="text-xs text-gray-500">The symbol used before text commands.</p>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">Log Channel ID</label>
            <input
              type="text"
              name="log_channel_id"
              value={config.log_channel_id}
              onChange={handleChange}
              placeholder="e.g. 1234567890"
              className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors"
            />
            <p className="text-xs text-gray-500">Channel for moderation logs.</p>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">Welcome Channel ID</label>
            <input
              type="text"
              name="welcome_channel_id"
              value={config.welcome_channel_id}
              onChange={handleChange}
              placeholder="e.g. 1234567890"
              className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors"
            />
            <p className="text-xs text-gray-500">Channel for greeting new members.</p>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">Counting Channel ID</label>
            <input
              type="text"
              name="counting_channel_id"
              value={config.counting_channel_id}
              onChange={handleChange}
              placeholder="e.g. 1234567890"
              className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors"
            />
            <p className="text-xs text-gray-500">Channel for the counting game.</p>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">Suggestion Channel ID</label>
            <input
              type="text"
              name="suggestion_channel_id"
              value={config.suggestion_channel_id}
              onChange={handleChange}
              placeholder="e.g. 1234567890"
              className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors"
            />
            <p className="text-xs text-gray-500">Channel for user suggestions.</p>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">Moderator Role ID</label>
            <input
              type="text"
              name="mod_role_id"
              value={config.mod_role_id}
              onChange={handleChange}
              placeholder="e.g. 1234567890"
              className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors"
            />
            <p className="text-xs text-gray-500">Role that bypasses some restrictions.</p>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">Auto-Role ID</label>
            <input
              type="text"
              name="auto_role_id"
              value={config.auto_role_id}
              onChange={handleChange}
              placeholder="e.g. 1234567890"
              className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-colors"
            />
            <p className="text-xs text-gray-500">Role assigned automatically on join.</p>
          </div>

        </div>

        <div className="pt-4 border-t border-gray-700 flex justify-end">
          <button
            type="submit"
            disabled={saving}
            className="flex items-center px-6 py-2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg font-medium transition-colors"
          >
            {saving ? (
              <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin mr-2"></div>
            ) : (
              <Save className="w-5 h-5 mr-2" />
            )}
            Save Changes
          </button>
        </div>
      </form>
    </div>
  );
}
