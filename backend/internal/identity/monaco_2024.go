package identity

import "time"

var weekendMappings = map[weekendLookup]WeekendMapping{
	{season: 2024, meetingKey: 1235}: {
		Season:              2024,
		SourceMeetingKey:    1235,
		MeetingCanonicalKey: "2024-monaco-grand-prix",
		CircuitCanonicalKey: "circuit-de-monaco",
		CircuitKey:          22,
		CircuitShortName:    "Monte Carlo",
		CountryCode:         "MON",
		CountryName:         "Monaco",
		Location:            "Monte Carlo",
		MeetingName:         "Monaco Grand Prix",
		MeetingOfficialName: "FORMULA 1 GRAND PRIX DE MONACO 2024",
		DateStart:           time.Date(2024, time.May, 24, 11, 30, 0, 0, time.UTC),
		DateEnd:             time.Date(2024, time.May, 26, 15, 0, 0, 0, time.UTC),
		IsCancelled:         false,
		Sessions: map[string]SessionMapping{
			"Practice 1": {Name: "Practice 1", Type: "Practice", CanonicalSuffix: "practice-1"},
			"Practice 2": {Name: "Practice 2", Type: "Practice", CanonicalSuffix: "practice-2"},
			"Practice 3": {Name: "Practice 3", Type: "Practice", CanonicalSuffix: "practice-3"},
			"Qualifying": {Name: "Qualifying", Type: "Qualifying", CanonicalSuffix: "qualifying"},
			"Race":       {Name: "Race", Type: "Race", CanonicalSuffix: "race"},
		},
	},
}

var driverMappings = map[driverLookup]DriverMapping{
	{2024, "Max VERSTAPPEN"}:   {CanonicalKey: "max-verstappen", DisplayName: "Max Verstappen", ExpectedAcronym: "VER", ExpectedNumber: 1},
	{2024, "Logan SARGEANT"}:   {CanonicalKey: "logan-sargeant", DisplayName: "Logan Sargeant", ExpectedAcronym: "SAR", ExpectedNumber: 2},
	{2024, "Daniel RICCIARDO"}: {CanonicalKey: "daniel-ricciardo", DisplayName: "Daniel Ricciardo", ExpectedAcronym: "RIC", ExpectedNumber: 3},
	{2024, "Lando NORRIS"}:     {CanonicalKey: "lando-norris", DisplayName: "Lando Norris", ExpectedAcronym: "NOR", ExpectedNumber: 4},
	{2024, "Pierre GASLY"}:     {CanonicalKey: "pierre-gasly", DisplayName: "Pierre Gasly", ExpectedAcronym: "GAS", ExpectedNumber: 10},
	{2024, "Sergio PEREZ"}:     {CanonicalKey: "sergio-perez", DisplayName: "Sergio Perez", ExpectedAcronym: "PER", ExpectedNumber: 11},
	{2024, "Fernando ALONSO"}:  {CanonicalKey: "fernando-alonso", DisplayName: "Fernando Alonso", ExpectedAcronym: "ALO", ExpectedNumber: 14},
	{2024, "Charles LECLERC"}:  {CanonicalKey: "charles-leclerc", DisplayName: "Charles Leclerc", ExpectedAcronym: "LEC", ExpectedNumber: 16},
	{2024, "Lance STROLL"}:     {CanonicalKey: "lance-stroll", DisplayName: "Lance Stroll", ExpectedAcronym: "STR", ExpectedNumber: 18},
	{2024, "Kevin MAGNUSSEN"}:  {CanonicalKey: "kevin-magnussen", DisplayName: "Kevin Magnussen", ExpectedAcronym: "MAG", ExpectedNumber: 20},
	{2024, "Yuki TSUNODA"}:     {CanonicalKey: "yuki-tsunoda", DisplayName: "Yuki Tsunoda", ExpectedAcronym: "TSU", ExpectedNumber: 22},
	{2024, "Alexander ALBON"}:  {CanonicalKey: "alexander-albon", DisplayName: "Alexander Albon", ExpectedAcronym: "ALB", ExpectedNumber: 23},
	{2024, "ZHOU Guanyu"}:      {CanonicalKey: "zhou-guanyu", DisplayName: "Zhou Guanyu", ExpectedAcronym: "ZHO", ExpectedNumber: 24},
	{2024, "Nico HULKENBERG"}:  {CanonicalKey: "nico-hulkenberg", DisplayName: "Nico Hulkenberg", ExpectedAcronym: "HUL", ExpectedNumber: 27},
	{2024, "Esteban OCON"}:     {CanonicalKey: "esteban-ocon", DisplayName: "Esteban Ocon", ExpectedAcronym: "OCO", ExpectedNumber: 31},
	{2024, "Lewis HAMILTON"}:   {CanonicalKey: "lewis-hamilton", DisplayName: "Lewis Hamilton", ExpectedAcronym: "HAM", ExpectedNumber: 44},
	{2024, "Carlos SAINZ"}:     {CanonicalKey: "carlos-sainz", DisplayName: "Carlos Sainz", ExpectedAcronym: "SAI", ExpectedNumber: 55},
	{2024, "George RUSSELL"}:   {CanonicalKey: "george-russell", DisplayName: "George Russell", ExpectedAcronym: "RUS", ExpectedNumber: 63},
	{2024, "Valtteri BOTTAS"}:  {CanonicalKey: "valtteri-bottas", DisplayName: "Valtteri Bottas", ExpectedAcronym: "BOT", ExpectedNumber: 77},
	{2024, "Oscar PIASTRI"}:    {CanonicalKey: "oscar-piastri", DisplayName: "Oscar Piastri", ExpectedAcronym: "PIA", ExpectedNumber: 81},
}

var constructorMappings = map[constructorLookup]ConstructorMapping{
	{2024, "Alpine"}:          {CanonicalKey: "2024-alpine", DisplayName: "Alpine"},
	{2024, "Aston Martin"}:    {CanonicalKey: "2024-aston-martin", DisplayName: "Aston Martin"},
	{2024, "Ferrari"}:         {CanonicalKey: "2024-ferrari", DisplayName: "Ferrari"},
	{2024, "Haas F1 Team"}:    {CanonicalKey: "2024-haas", DisplayName: "Haas F1 Team"},
	{2024, "Kick Sauber"}:     {CanonicalKey: "2024-kick-sauber", DisplayName: "Kick Sauber"},
	{2024, "McLaren"}:         {CanonicalKey: "2024-mclaren", DisplayName: "McLaren"},
	{2024, "Mercedes"}:        {CanonicalKey: "2024-mercedes", DisplayName: "Mercedes"},
	{2024, "RB"}:              {CanonicalKey: "2024-rb", DisplayName: "RB"},
	{2024, "Red Bull Racing"}: {CanonicalKey: "2024-red-bull-racing", DisplayName: "Red Bull Racing"},
	{2024, "Williams"}:        {CanonicalKey: "2024-williams", DisplayName: "Williams"},
}
