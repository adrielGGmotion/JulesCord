CREATE TABLE IF NOT EXISTS stocks (
    symbol TEXT PRIMARY KEY,
    current_price INT NOT NULL DEFAULT 100,
    history JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE TABLE IF NOT EXISTS user_stocks (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    symbol TEXT NOT NULL REFERENCES stocks(symbol),
    quantity INT NOT NULL DEFAULT 0,
    average_buy_price FLOAT NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, user_id, symbol)
);

-- Seed some initial stocks
INSERT INTO stocks (symbol, current_price, history) VALUES
    ('JULS', 150, '[150]'::jsonb),
    ('DOGE', 50, '[50]'::jsonb),
    ('STONK', 200, '[200]'::jsonb),
    ('MEOW', 75, '[75]'::jsonb)
ON CONFLICT (symbol) DO NOTHING;
