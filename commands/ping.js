const { SlashCommandBuilder } = require('discord.js');

module.exports = {
    data: new SlashCommandBuilder()
        .setName('ping')
        .setDescription('Replies with Pong! Jules style.'),
    async execute(interaction) {
        // Calculate round-trip latency
        const sent = await interaction.reply({ content: 'Pinging...', fetchReply: true });
        const roundtripLatency = sent.createdTimestamp - interaction.createdTimestamp;

        await interaction.editReply(`Pong! 🏓\nLatency is ${roundtripLatency}ms.\nAPI Latency is ${Math.round(interaction.client.ws.ping)}ms.\n\nNot bad for a bot that built itself, right?`);
    },
};
