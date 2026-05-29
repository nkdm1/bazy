CREATE TABLE `users` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `email` varchar(255) UNIQUE NOT NULL,
  `password_hash` varchar(255),
  `name` varchar(100) NOT NULL,
  `surname` varchar(100) NOT NULL,
  `role` ENUM ('admin', 'referee', 'viewer') NOT NULL DEFAULT 'viewer',
  `created_at` timestamp NOT NULL DEFAULT (now()),
  `deleted_at` timestamp
);

CREATE TABLE `set_mail` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `user_id` integer NOT NULL,
  `new_mail` varchar(255) NOT NULL,
  `token_hash` varchar(255) UNIQUE NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT (now()),
  `expire_time` timestamp NOT NULL DEFAULT ((NOW() + INTERVAL 1 HOUR)),
  `status` ENUM ('pending', 'expired', 'used') NOT NULL DEFAULT 'pending'
);

CREATE TABLE `set_password` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `user_id` integer NOT NULL,
  `token_hash` varchar(255) UNIQUE NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT (now()),
  `expire_time` timestamp NOT NULL DEFAULT ((NOW() + INTERVAL 1 HOUR)),
  `status` ENUM ('pending', 'expired', 'used') NOT NULL DEFAULT 'pending'
);

CREATE TABLE `referees` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `address_id` integer NOT NULL,
  `user_id` integer UNIQUE NOT NULL,
  `phone` varchar(255)
);

CREATE TABLE `set_phone` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `referee_id` integer NOT NULL,
  `new_phone` varchar(255) NOT NULL,
  `token_hash` varchar(255) UNIQUE NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT (now()),
  `expire_time` timestamp NOT NULL DEFAULT ((NOW() + INTERVAL 1 HOUR)),
  `status` ENUM ('pending', 'expired', 'used') NOT NULL DEFAULT 'pending'
);

CREATE TABLE `address` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `postcode` varchar(6) NOT NULL,
  `city` varchar(100) NOT NULL,
  `street` varchar(100),
  `street_number` varchar(100) NOT NULL,
  `flat_number` varchar(100)
);

CREATE TABLE `licenses` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `referee_id` integer NOT NULL,
  `license_number` varchar(255) UNIQUE NOT NULL,
  `license_name_id` integer NOT NULL,
  `issued_at` date NOT NULL,
  `expire_at` date NOT NULL
);

CREATE TABLE `licenses_names` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `license_name` varchar(255) NOT NULL
);

CREATE TABLE `availability` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `referee_id` integer NOT NULL,
  `available_date` date NOT NULL
);

CREATE TABLE `teams` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `city` varchar(100) NOT NULL
);

CREATE TABLE `matches` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `match_start` timestamp NOT NULL,
  `match_end` timestamp NOT NULL,
  `level_of_match` integer NOT NULL,
  `venue_id` integer NOT NULL,
  `home_team_id` integer NOT NULL,
  `away_team_id` integer NOT NULL,
  `status` ENUM ('scheduled', 'completed', 'cancelled') NOT NULL DEFAULT 'scheduled',
  `home_team_points` integer,
  `away_team_points` integer
);

CREATE TABLE `match_assignments` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `referee_id` integer NOT NULL,
  `match_id` integer NOT NULL,
  `role` integer NOT NULL,
  `assignment_status` ENUM ('pending', 'accepted', 'declined', 'cancelled', 'noshow') NOT NULL DEFAULT 'pending'
);

CREATE TABLE `wages` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `match_level` integer NOT NULL,
  `role_in_match` integer NOT NULL,
  `fee` decimal(10,2) NOT NULL,
  `valid_from` date NOT NULL
);

CREATE TABLE `matches_level` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `match_level` ENUM ('fiba', 'plk', 'centralna', 'okregowa', 'stazysta') NOT NULL
);

CREATE TABLE `role_in_match` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `match_role` ENUM ('crew_chief', 'umpire') NOT NULL DEFAULT 'umpire'
);

CREATE TABLE `venues` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `gym_name` varchar(255) NOT NULL,
  `address_id` integer NOT NULL
);

CREATE TABLE `payouts` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `assignment_id` integer UNIQUE NOT NULL,
  `wages_id` integer NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `status` ENUM ('pending', 'sent', 'paid', 'failed') NOT NULL DEFAULT 'pending',
  `bank_transaction_id` varchar(255),
  `paid_at` timestamp
);

CREATE TABLE `reviews` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `referee_id` integer NOT NULL,
  `match_id` integer NOT NULL,
  `rating` integer NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT (now()),
  `created_by` integer NOT NULL
);

CREATE TABLE `auth_tokens` (
  `id` integer PRIMARY KEY AUTO_INCREMENT,
  `user_id` integer NOT NULL,
  `token_hash` varchar(255) UNIQUE NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT (now()),
  `expire_time` timestamp NOT NULL DEFAULT ((NOW() + INTERVAL 1 HOUR)),
  `last_used_at` timestamp
);

CREATE UNIQUE INDEX `availability_index_0` ON `availability` (`referee_id`, `available_date`);

ALTER TABLE `auth_tokens` ADD FOREIGN KEY (`user_id`) REFERENCES `users` (`id`);

ALTER TABLE `referees` ADD FOREIGN KEY (`user_id`) REFERENCES `users` (`id`);

ALTER TABLE `licenses` ADD FOREIGN KEY (`referee_id`) REFERENCES `referees` (`id`);

ALTER TABLE `availability` ADD FOREIGN KEY (`referee_id`) REFERENCES `referees` (`id`);

ALTER TABLE `matches` ADD FOREIGN KEY (`home_team_id`) REFERENCES `teams` (`id`);

ALTER TABLE `matches` ADD FOREIGN KEY (`away_team_id`) REFERENCES `teams` (`id`);

ALTER TABLE `matches` ADD FOREIGN KEY (`venue_id`) REFERENCES `venues` (`id`);

ALTER TABLE `match_assignments` ADD FOREIGN KEY (`match_id`) REFERENCES `matches` (`id`);

ALTER TABLE `match_assignments` ADD FOREIGN KEY (`referee_id`) REFERENCES `referees` (`id`);

ALTER TABLE `payouts` ADD FOREIGN KEY (`assignment_id`) REFERENCES `match_assignments` (`id`);

ALTER TABLE `reviews` ADD FOREIGN KEY (`referee_id`) REFERENCES `referees` (`id`);

ALTER TABLE `reviews` ADD FOREIGN KEY (`match_id`) REFERENCES `matches` (`id`);

ALTER TABLE `set_mail` ADD FOREIGN KEY (`user_id`) REFERENCES `users` (`id`);

ALTER TABLE `set_password` ADD FOREIGN KEY (`user_id`) REFERENCES `users` (`id`);

ALTER TABLE `reviews` ADD FOREIGN KEY (`created_by`) REFERENCES `users` (`id`);

ALTER TABLE `matches` ADD FOREIGN KEY (`level_of_match`) REFERENCES `matches_level` (`id`);

ALTER TABLE `wages` ADD FOREIGN KEY (`match_level`) REFERENCES `matches_level` (`id`);

ALTER TABLE `match_assignments` ADD FOREIGN KEY (`role`) REFERENCES `role_in_match` (`id`);

ALTER TABLE `wages` ADD FOREIGN KEY (`role_in_match`) REFERENCES `role_in_match` (`id`);

ALTER TABLE `payouts` ADD FOREIGN KEY (`wages_id`) REFERENCES `wages` (`id`);

ALTER TABLE `licenses` ADD FOREIGN KEY (`license_name_id`) REFERENCES `licenses_names` (`id`);

ALTER TABLE `venues` ADD FOREIGN KEY (`address_id`) REFERENCES `address` (`id`);

ALTER TABLE `referees` ADD FOREIGN KEY (`address_id`) REFERENCES `address` (`id`);

ALTER TABLE `set_phone` ADD FOREIGN KEY (`referee_id`) REFERENCES `referees` (`id`);
