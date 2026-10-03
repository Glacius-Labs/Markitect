# Task 1: People represented by meeting attendees

Add the `GetMeetingPeopleCount` query to the Meetings Application. It accepts a `MeetingId` and returns the number of people represented by the rows returned for that meeting by the existing `GetMeetingAttendees` query. Count one person for each returned attendee row, then add that row's guest count. Treat a null guest count as zero. Return only the integer result; do not add an HTTP endpoint.
