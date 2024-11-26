# Installation
Installation nécessaire pour utilisé stockfish:

Sur mac: 
```
brew install stockfish
```

Autres : [stockfish](https://stockfishchess.org/download/)


# Utilisation

Pour lancer le serveur, il suffit de se déplacer dans le répertoire du serveur et d'exécuter cette commande:

```
go run .
```

Pour lancer un client, il suffit de se déplacer dans le répertoire client et d'exécuter cette commande:

```
go run . protocol port nom
```

exemple:
```
go run . udp 8001 client1
```

| protocol  | port  |
| --------- | --------- |
| udp | 8001 |
| tcp | 8000|

