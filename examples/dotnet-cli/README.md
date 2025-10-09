# .NET CLI Example

A simple .NET 6.0 CLI application that runs a minimal web server.

## Requirements

- .NET 6.0 SDK

## Running locally

```bash
dotnet restore
dotnet run
```

The application will start a web server on port 3000 (or the port specified by ASPNETCORE_URLS).

## Build

```bash
dotnet publish -c Release -o out
```

