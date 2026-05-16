1 - Get all upcoming matches with team names and venue details:
SELECT m.match_date, t_home.name AS home_team, t_away.name AS away_team, v.gym_name 
FROM matches m
JOIN teams t_home ON m.home_team_id = t_home.id
JOIN teams t_away ON m.away_team_id = t_away.id
JOIN venues v ON m.venue_id = v.id
WHERE m.match_date > NOW() 
ORDER BY m.match_date ASC;

2 - Find all referees assigned to a specific match: 
SELECT r.name, ma.role, ma.assignment_status 
FROM match_assignments ma
JOIN referees r ON ma.referee_id = r.id
WHERE ma.match_id = 5;

3 - List upcoming matches for a specific referee (using their User ID):
SELECT m.match_date, ma.role, ma.assignment_status
FROM match_assignments ma
JOIN matches m ON ma.match_id = m.id
JOIN referees r ON ma.referee_id = r.id
WHERE r.user_id = 10 AND m.match_date > NOW();

4 - Find referees who declared themselves available for a specific date:
SELECT r.name, r.phone 
FROM availability a
JOIN referees r ON a.referee_id = r.id
WHERE a.available_date = '2026-05-15' AND a.is_available = TRUE;

5 - Update assignment status to 'accepted' when a referee accepts a timeslot:
UPDATE match_assignments 
SET assignment_status = 'accepted' 
WHERE match_id = 12 AND referee_id = 3;

6 - Calculate total amount earned by a referee:
SELECT r.name, SUM(p.amount) AS total_earned
FROM payouts p
JOIN referees r ON p.referee_id = r.id
WHERE p.status = 'Paid'
GROUP BY r.name;

7 - List all pending payouts that need to be processed by an Admin:
SELECT p.id, r.name, p.amount, m.match_date
FROM payouts p
JOIN referees r ON p.referee_id = r.id
JOIN matches m ON p.match_id = m.id
WHERE p.status = 'Pending';

8 - Count how many matches each referee has been assigned to:
SELECT r.name, COUNT(ma.id) AS matches_count
FROM referees r
LEFT JOIN match_assignments ma ON r.id = ma.referee_id
GROUP BY r.id, r.name;

9 - Get a referees license level based on their email login:
SELECT r.name, l.level_name
FROM users u
JOIN referees r ON u.id = r.user_id
JOIN licenses l ON r.license_id = l.id
WHERE u.email = 'referee@example.com';

10 - Find matches that currently have no referees assigned:
SELECT m.id, m.match_date, t_home.name, t_away.name
FROM matches m
LEFT JOIN match_assignments ma ON m.id = ma.match_id
JOIN teams t_home ON m.home_team_id = t_home.id
JOIN teams t_away ON m.away_team_id = t_away.id
WHERE ma.id IS NULL;

11 - Mark a match as 'Completed' and update its status:
UPDATE matches 
SET status = 'Completed' 
WHERE id = 45;

12 - List all referees who have 'declined' an assignment:
SELECT r.name, m.match_date, ma.role
FROM match_assignments ma
JOIN referees r ON ma.referee_id = r.id
JOIN matches m ON ma.match_id = m.id
WHERE ma.assignment_status = 'declined';

13 - Get all teams from a specific city:
SELECT name FROM teams WHERE city = 'Wrocław';

14 - Show the most used venue for matches:
SELECT v.gym_name, COUNT(m.id) AS usage_count
FROM venues v
JOIN matches m ON v.id = m.venue_id
GROUP BY v.id, v.gym_name
ORDER BY usage_count DESC
LIMIT 1;

15 - Filter matches for the upcoming week:
SELECT * FROM matches 
WHERE match_date BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL 7 DAY);

16 - Check if a specific referee is available before assigning them:
SELECT EXISTS (
  SELECT 1 FROM availability 
  WHERE referee_id = 4 AND available_date = '2026-06-20' AND is_available = TRUE
) AS can_be_assigned;

17 - Get a summary of how many games each team has played:
SELECT t.name, COUNT(m.id) AS games_played
FROM teams t
JOIN matches m ON t.id = m.home_team_id OR t.id = m.away_team_id
WHERE m.status = 'Completed'
GROUP BY t.id, t.name;

18 - Register a new availability slot for a referee:
INSERT INTO availability (referee_id, available_date, is_available) 
VALUES (2, '2026-07-01', TRUE);

19 - List all referees and their contact info for a specific license level:
SELECT r.name, r.phone, r.email
FROM referees r
JOIN licenses l ON r.license_id = l.id
WHERE l.level_name = 'NBA Certified';

20 - Calculate the total payout budget spent for a specific month:
SELECT SUM(amount) AS monthly_budget
FROM payouts
WHERE paid_at BETWEEN '2026-01-01' AND '2026-01-31';
