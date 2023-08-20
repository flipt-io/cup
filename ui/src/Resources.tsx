import * as React from 'react';
import { useEffect } from 'react';
import ndjsonStream from 'can-ndjson-stream';
import { useParams } from 'react-router-dom';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table';
import { Badge } from './components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './components/ui/tabs';
import { TableIcon, DashboardIcon } from '@radix-ui/react-icons'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './components/ui/card';

const loopThroughNDJSON = async (reader: any): Promise<any[]> => {
  let done = false;
  let results = [];
  while (!done) {
    const { value, done: _done } = await reader.read();
    done = _done;
    if (value) {
      results.push(value);
    }
  }
  return results;
};

const Resources: React.FunctionComponent<any> = () => {
  let { group, version, kind, namespace } = useParams<{
    group: string;
    version: string;
    namespace: string;
    kind: string;
  }>();
  const [resources, setResources] = React.useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      let response = await fetch(
        `http://localhost:8181/apis/${group}/${version}/namespaces/${namespace}/${kind}`
      );
      let reader = ndjsonStream(response.body).getReader();
      let results = await loopThroughNDJSON(reader);
      setResources(results);
    };

    fetchData();
  }, []);

  if (!resources) {
    return null;
  }

  return (
    <>
      <h1 className="flex scroll-m-20 border-b pb-2 mb-4 text-3xl font-semibold tracking-tight transition-colors first:mt-0">
        <span className="self-center text-muted-foreground">{group}/</span>
        <span className="self-center mr-2">{kind}</span>
        <Badge className="self-center" variant="secondary">
          {version}
        </Badge>
      </h1>
      <Tabs defaultValue="account" className="w-full">
        <TabsList className='mb-4'>
          <TabsTrigger value="table"><TableIcon /></TabsTrigger>
          <TabsTrigger value="cards"><DashboardIcon /></TabsTrigger>
        </TabsList>
        <TabsContent value="table">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Namespace</TableHead>
                <TableHead>Name</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {resources.map((resource) => {
                  return (
                  <TableRow
                    key={`row/${resource.metadata.namespace}/${resource.metadata.name}`}
                  >
                    <TableCell className="text-left">
                      {resource.metadata.namespace}
                    </TableCell>
                    <TableCell className="text-left">
                      {resource.metadata.name}
                    </TableCell>
                    </TableRow>
                  );
              })}
            </TableBody>
          </Table>
        </TabsContent>
        <TabsContent value="cards">
          <div className='flex'>
            {
              resources.map((resource) => {
                return (
                  <Card className='w-1/3 mr-auto text-left' key={`card/${resource.metadata.namespace}/${resource.metadata.name}`}>
                    <CardHeader>
                      <CardTitle>
                        <span className='text-muted-foreground'>{resource.metadata.namespace}/</span>
                        {resource.metadata.name}
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <p>Card Content</p>
                    </CardContent>
                  </Card>
                );
              })
            }
          </div>
        </TabsContent>
      </Tabs>
    </>
  );
};

export default Resources;
