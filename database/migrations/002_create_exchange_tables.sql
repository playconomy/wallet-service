-- Exchange rates table
CREATE TABLE exchange_rates (
    id SERIAL PRIMARY KEY,
    game_id VARCHAR(50) NOT NULL,
    token_type VARCHAR(20) NOT NULL,
    to_platform_ratio NUMERIC(10, 4) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(game_id, token_type)
);

-- Wallet logs table
CREATE TABLE wallet_logs (
    id SERIAL PRIMARY KEY,
    wallet_id INT NOT NULL,
    user_id INT NOT NULL,
    game_id VARCHAR(50),
    token_type VARCHAR(20),
    amount NUMERIC(20, 2) NOT NULL,
    platform_amount NUMERIC(20, 2) NOT NULL,
    source VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (wallet_id) REFERENCES wallets(id)
);
