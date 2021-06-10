# BBStruction
[Request from sensenstahl 06/06/21](https://discord.com/channels/@me/808676590351024158/850992251408482336)

### Original request

- tool makes a table (hash values) of all files in a given directory (the db)
  - maybe doing so with a white/blacklist or simply "all"-option for files?
    and maybe 3-4 sub dirs deep
  - switch to do a new indexing only without running the tool afterwards
  - switch to do a new indexing and running the tool afterwards
  - switch to run the tool if there is already a table present
    (like in the case there is no data base update)

- tool checks a given directory (maybe e.g. 3-4 sub dirs deep?)
  comparing the files
  - if files are identical in content based on the
    index table then delete those files
  - white/blacklist for deleting no matter any switches?

- if a directory is empty after a check delete that directory as well
  (maybe white/blacklist if there is NO filetype present based on
  file extension then delete a directory no matter the left content)

- goal is to scrap all unneeded data so you have to look at less content
  since you don't get to see stuff that is already in your indexed db
  (does need 1:1 visual comparing if there is e.g. data rot/typo in a scroller/
  hidden vs. clear number)

- white/blacklist(s) in an extra .txt file that can be edited easily

- or maybe not deleting files but copy all potential remaining data keeping the
  structure into a main directory so the original (e.g. unpacked) stuff is
  available in case something needs to get checked again by hand so it is not
  needed to extract a .zip again. thoughts? maybe a switch for that?
  
