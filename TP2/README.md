# Installation
Installation nécessaire pour utilisé stockfish:

Sur mac: 
```
brew install stockfish
```

Autres : [stockfish](https://stockfishchess.org/download/)


# Lancement

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
go run . udp 8081 client1
```

| protocol  | port  |
| --------- | --------- |
| udp | 8081 |
| tcp | 8080|

# Menu de jeu

Après l'authentification du client, trois choix s'offrent au joueur

- play :
    ```
    1 - Solo
    2 - Multijoueur
    ```
    - 1 -> Partie solo contre stockfish, la partie commence directement.

    - 2 -> Un uuid de partie est envoyer au client pour le partager à son adversaire, la partie commence seulement lorsqe l'adversaire rejoint la partie.

- join :
    ```
    1 - Rejoindre quelqu'un
    2 - Charger une ancienne partie
    ```

    - 1 -> Affiche une message indiquant d'envoyer le uuid de la partie à rejoindre.

    - 2 -> Affiche toute les parties qui ne sont pas terminée où le client est un joueur.

- exit : Quitte l'application
    
# Jouer

Pour jouer un coup, il suffit d'écrire la position de la pièce que nous souhaitons déplacer, par exemple: a2, suivi de la position de destination, par exemple a4. 
Exemple:

```
 A B C D E F G H
8♜ ♞ ♝ ♛ ♚ ♝ ♞ ♜ 
7♟ ♟ ♟ ♟ ♟ ♟ ♟ ♟ 
6- - - - - - - - 
5- - - - - - - - 
4- - - - - - - - 
3- - - - - - - - 
2♙ ♙ ♙ ♙ ♙ ♙ ♙ ♙ 
1♖ ♘ ♗ ♕ ♔ ♗ ♘ ♖ 

> a2a4
```
Résultat: 

```
 A B C D E F G H
8♜ ♞ ♝ ♛ ♚ ♝ ♞ ♜ 
7♟ ♟ ♟ ♟ ♟ ♟ ♟ ♟ 
6- - - - - - - - 
5- - - - - - - - 
4♙ - - - - - - - 
3- - - - - - - - 
2- ♙ ♙ ♙ ♙ ♙ ♙ ♙ 
1♖ ♘ ♗ ♕ ♔ ♗ ♘ ♖ 
```

**Note** : *En solo, le joueur commence toujours le premier et en multijouer le joueur qui a créer la partie commence en premier.*
