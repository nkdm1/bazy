INSERT INTO address (id, postcode, city, street, flat_number, street_number) VALUES
(105832, '03-901', 'Warszawa', 'al. Księcia Józefa Poniatowskiego', NULL, '1'),
(294711, '54-118', 'Wrocław', 'al. Śląska', NULL, '1'),
(381920, '09-400', 'Płock', 'ul. Ignacego Łukasiewicza', 12, '34'),
(472839, '02-528', 'Warszawa', 'ul. Rakowiecka', 5, '2A');

INSERT INTO users (id, name, surname, role) VALUES
(509211, 'Tomasz', 'Kwiatkowski', 'referee'),
(618392, 'Daniel', 'Stefański', 'referee'),
(729403, 'Adam', 'Wiśniewski', 'admin');

INSERT INTO teams (id, name, city) VALUES
(810293, 'Legia Warszawa', 'Warszawa'),
(821304, 'Śląsk Wrocław', 'Wrocław'),
(832415, 'Lech Poznań', 'Poznań'),
(843526, 'Jagiellonia Białystok', 'Białystok');

INSERT INTO venues (id, gym_name, address_id) VALUES
(914253, 'PGE Narodowy', 105832),
(925364, 'Tarczyński Arena Wrocław', 294711);

INSERT INTO matches_level (id, match_level) VALUES
(110293, 'plk'),
(121304, 'centralna');

INSERT INTO licenses_names (id, license_name) VALUES
(319283, 'Licencja PZPN Ekstraklasa / VAR'),
(320394, 'Licencja PZPN Szczebel Centralny');

INSERT INTO referees (id, user_id, address_id, phone, status) VALUES
(418273, 509211, 381920, '+48600111222', 'active'),
(429384, 618392, 472839, '+48500333444', 'active');

INSERT INTO mail_verification (id, user_id, mail_given, verification_code, send_time, expire_time, is_verifed) VALUES
(517263, 509211, 't.kwiatkowski@pzpn.pl', '258491', '2026-05-30 12:00:00', '2026-06-01 12:00:00', true),
(528374, 618392, 'd.stefanski@pzpn.pl', '901283', '2026-06-05 10:00:00', '2026-06-05 23:59:59', false);

INSERT INTO set_password (id, reset_token, password_hash, created_at, expire_time, status) VALUES
(509211, 'f47ac10b-58cc-4372-a567-0e02b2c3d479', '$2b$12$eImiTXuWVxfM37uY4JANjQ==', '2026-05-01 10:00:00', '2026-05-15 14:00:00', 'used'),
(618392, '550e8400-e29b-41d4-a716-446655440000', '$2b$12$Kjh398dsa98321mds0921M==', '2026-06-01 09:00:00', '2026-06-02 20:00:00', 'active');

INSERT INTO wages (id, match_level, role_in_match, fee, valid_from) VALUES
(715243, 110293, 210984, 4200.00, '2026-01-01'),
(726354, 121304, 221095, 2100.00, '2026-01-01');

INSERT INTO matches (id, match_date, match_end, level_of_match, venue_id, home_team_id, away_team_id, status, home_team_points, away_team_points) VALUES
(814233, '2026-05-25 18:00:00', '2026-05-25 20:00:00', 110293, 925364, 821304, 810293, 'Completed', 85, 91),
(825344, '2026-06-15 20:30:00', '2026-06-15 22:30:00', 110293, 914253, 810293, 832415, 'Scheduled', NULL, NULL);

INSERT INTO licenses (id, referees_id, license_number, license_name, issued_at, expire_at) VALUES
(913223, 418273, 'KS/PZPN/2026/015', 319283, '2026-01-01', '2026-12-31'),
(924334, 429384, 'KS/PZPN/2026/028', 319283, '2026-01-01', '2026-12-31');

INSERT INTO availability (id, referee_id, available_date) VALUES
(102301, 418273, '2026-06-01'),
(102302, 418273, '2026-06-02'),
(102303, 418273, '2026-06-03'),
(102304, 418273, '2026-06-04'),
(102305, 418273, '2026-06-05'),
(102306, 418273, '2026-06-06'),
(102307, 418273, '2026-06-07'),
(113420, 429384, '2026-06-20'),
(113421, 429384, '2026-06-21'),
(113422, 429384, '2026-06-22'),
(113423, 429384, '2026-06-23'),
(113424, 429384, '2026-06-24'),
(113425, 429384, '2026-06-25');

INSERT INTO match_assignments (id, referee_id, match_id, role_in_match, assignment_status, created_at) VALUES
(201001, 418273, 814233, 'Crew_Chief', 'accepted', '2026-05-18 14:00:00'),
(201002, 429384, 814233, 'Umpire', 'accepted', '2026-05-18 14:05:00'),
(201201, 418273, 825344, 'Crew_Chief', 'accepted', '2026-05-18 14:00:00'),
(212312, 429384, 825344, 'Umpire', 'pending', '2026-05-18 14:05:00');

INSERT INTO payouts (id, assignment_id, wages_id, status, amount, paid_at) VALUES
(309190, 201001, 715243, 'Paid', 4200.00, '2026-05-28 12:45:00'),
(310201, 201002, 726354, 'Paid', 2100.00, '2026-05-28 12:45:00');

INSERT INTO reviews (id, referee_id, rating, created_by, created_at) VALUES
(408189, 418273, 8, 729403, '2026-05-26 10:15:00'),
(419290, 429384, 8, 729403, '2026-05-26 10:30:00');
