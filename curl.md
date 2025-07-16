curl -X POST http://localhost:3000/create   -H "Content-Type: application/json"   -d '{
    "url": "https://example.com",
    "slug": "mein-slug",
    "exp": "2025-12-31",
    "title": "Beispieltitel",
    "desc": "Dies ist eine Beschreibung."
}'